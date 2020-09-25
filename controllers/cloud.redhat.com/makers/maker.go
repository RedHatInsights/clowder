/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package makers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//DependencyMaker makes the DependencyConfig object
type DependencyMaker struct {
	*Maker
	config config.DependenciesConfig
}

//Maker struct for passing variables into SubMakers
type Maker struct {
	App     *crd.ClowdApp
	Env     *crd.ClowdEnvironment
	Client  client.Client
	Ctx     context.Context
	Request *ctrl.Request
	Log     logr.Logger
}

type makerUpdater struct {
	maker  *Maker
	update utils.Updater
}

// MakerUpdater encapsulates saving an object
type MakerUpdater interface {
	Apply(obj runtime.Object) (ctrl.Result, error)
	Updater() utils.Updater
}

func (mu *makerUpdater) Apply(obj runtime.Object) (ctrl.Result, error) {
	return mu.update.Apply(mu.maker.Ctx, mu.maker.Client, obj)
}

func (mu *makerUpdater) Updater() utils.Updater {
	return mu.update
}

// GetNamespacedName returns a new types.NamespacedName from the request based
// on pattern
func GetNamespacedName(r *reconcile.Request, pattern, namespace string) types.NamespacedName {
	if namespace == "" {
		namespace = r.Namespace
	}
	return types.NamespacedName{
		Name:      fmt.Sprintf(pattern, r.Name),
		Namespace: namespace,
	}
}

// Get is a convenience wrapper for common upsert queries
func (m *Maker) Get(nn types.NamespacedName, obj runtime.Object) (MakerUpdater, error) {
	err := m.Client.Get(m.Ctx, nn, obj)
	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return nil, err
	}
	return &makerUpdater{
		maker:  m,
		update: update,
	}, nil
}

// MakeLabeler creates a function that will label objects with metadata from
// the given namespaced name and labels
func (m *Maker) MakeLabeler(nn types.NamespacedName, labels map[string]string) func(metav1.Object) {
	return func(o metav1.Object) {
		o.SetName(nn.Name)
		o.SetNamespace(nn.Namespace)
		o.SetLabels(labels)
		o.SetOwnerReferences([]metav1.OwnerReference{m.Env.MakeOwnerReference()})
	}
}

func New(maker *Maker) (*Maker, error) {
	return maker, nil
}

//Make generates objects and dependencies for operator
func (m *Maker) Make() (ctrl.Result, error) {
	result, c, err := m.runProviders()

	if result, err := m.makeDependencies(c); err != nil {
		return result, err
	}

	hash, result, err := m.persistConfig(c)

	if err != nil {
		return result, err
	}

	if result, err := m.makeDeployment(hash); err != nil {
		return result, err
	}

	if result, err := m.makeService(); err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (m *Maker) makeService() (ctrl.Result, error) {

	result := ctrl.Result{}
	s := core.Service{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &s)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return result, err
	}

	ports := []core.ServicePort{
		{Name: "metrics", Port: m.Env.Spec.Metrics.Port, Protocol: "TCP"},
	}

	if m.App.Spec.Web == true {
		webPort := core.ServicePort{Name: "web", Port: m.Env.Spec.Web.Port, Protocol: "TCP"}
		ports = append(ports, webPort)
	}

	m.App.SetObjectMeta(&s)
	s.Spec.Selector = m.App.GetLabels()
	s.Spec.Ports = ports

	return update.Apply(m.Ctx, m.Client, &s)
}

func (m *Maker) persistConfig(c *config.AppConfig) (string, ctrl.Result, error) {

	result := ctrl.Result{}
	// In any case, we want to overwrite the secret, so this just
	// tests to see if the secret exists
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &core.Secret{})

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return "", result, err
	}

	jsonData, err := json.Marshal(c)
	if err != nil {
		return "", result, err
	}

	h := sha256.New()
	h.Write([]byte(jsonData))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	secret := core.Secret{
		StringData: map[string]string{
			"cdappconfig.json": string(jsonData),
		},
	}

	m.App.SetObjectMeta(&secret)

	result, err = update.Apply(m.Ctx, m.Client, &secret)
	return hash, result, err
}

func (m *Maker) getConfig() (*config.AppConfig, error) {
	secret := core.Secret{}
	appConfig := config.AppConfig{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &secret)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(secret.Data["cdappconfig.json"]), &appConfig)
	if err != nil {
		return nil, err
	}
	return &appConfig, nil
}

// This should probably take arguments for addtional volumes, so that we can add those and then do one Apply
func (m *Maker) makeDeployment(hash string) (ctrl.Result, error) {

	result := ctrl.Result{}
	d := apps.Deployment{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &d)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return result, err
	}

	m.App.SetObjectMeta(&d)

	d.Spec.Replicas = m.App.Spec.MinReplicas
	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: m.App.GetLabels()}
	d.Spec.Template.ObjectMeta.Labels = m.App.GetLabels()

	d.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
		{Name: "quay-cloudservices-pull"},
	}

	env := m.App.Spec.Env
	env = append(env, core.EnvVar{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"})

	var livenessProbe core.Probe
	var readinessProbe core.Probe

	baseProbe := core.Probe{
		Handler: core.Handler{
			HTTPGet: &core.HTTPGetAction{
				Path:   "/healthz",
				Scheme: "HTTP",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: m.Env.Spec.Web.Port,
				},
			},
		},
		FailureThreshold:    3,
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		TimeoutSeconds:      1,
	}
	if m.App.Spec.LivenessProbe != nil {
		livenessProbe = *m.App.Spec.LivenessProbe
	} else if m.App.Spec.Web {
		livenessProbe = baseProbe
	}
	if m.App.Spec.ReadinessProbe != nil {
		readinessProbe = *m.App.Spec.ReadinessProbe
	} else {
		readinessProbe = baseProbe
		readinessProbe.InitialDelaySeconds = 45

	}

	c := core.Container{
		Name:         m.App.ObjectMeta.Name,
		Image:        m.App.Spec.Image,
		Command:      m.App.Spec.Command,
		Args:         m.App.Spec.Args,
		Env:          env,
		Resources:    m.App.Spec.Resources,
		VolumeMounts: m.App.Spec.VolumeMounts,
		Ports: []core.ContainerPort{{
			Name:          "metrics",
			ContainerPort: m.Env.Spec.Metrics.Port,
		}},
		ImagePullPolicy: core.PullIfNotPresent,
	}

	if (core.Probe{}) != livenessProbe {
		c.LivenessProbe = &livenessProbe
	}
	if (core.Probe{}) != readinessProbe {
		c.ReadinessProbe = &readinessProbe
	}

	if m.App.Spec.Web {
		c.Ports = append(c.Ports, core.ContainerPort{
			Name:          "web",
			ContainerPort: m.Env.Spec.Web.Port,
		})
	}

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "config-secret",
		MountPath: "/cdapp/",
	})

	d.Spec.Template.Spec.Containers = []core.Container{c}

	d.Spec.Template.Spec.Volumes = m.App.Spec.Volumes
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName: m.App.ObjectMeta.Name,
			},
		},
	})

	annotations := make(map[string]string)
	annotations["configHash"] = hash
	d.Spec.Template.SetAnnotations(annotations)
	if result, err := update.Apply(m.Ctx, m.Client, &d); err != nil {
		return result, err
	}

	return result, nil
}

func (m *Maker) runProviders() (ctrl.Result, *config.AppConfig, error) {
	result := ctrl.Result{}

	provider := providers.Provider{
		Client: m.Client,
		Ctx:    m.Ctx,
		Env:    m.Env,
	}

	c := config.AppConfig{}

	objectStoreProvider, err := provider.GetObjectStore()

	if err != nil {
		return result, &c, err
	}

	for _, bucket := range m.App.Spec.ObjectStore {
		objectStoreProvider.CreateBucket(bucket)
	}

	objectStoreProvider.Configure(&c)

	dbSpec := m.App.Spec.Database

	if dbSpec.Name != "" {
		databaseProvider, err := provider.GetDatabase()

		if err != nil {
			return result, &c, err
		}

		databaseProvider.CreateDatabase(m.App)
		databaseProvider.Configure(&c)
	}

	kafkaProvider, err := provider.GetKafka()

	if err != nil {
		return result, &c, err
	}

	nn := types.NamespacedName{
		Name:      m.App.Name,
		Namespace: m.App.Namespace,
	}

	for _, topic := range m.App.Spec.KafkaTopics {
		kafkaProvider.CreateTopic(nn, &topic)

		if err != nil {
			return result, &c, err
		}
	}

	kafkaProvider.Configure(&c)

	if m.App.Spec.InMemoryDB {
		inMemoryDbProvider, err := provider.GetInMemoryDB()

		if err != nil {
			return result, &c, err
		}

		inMemoryDbProvider.CreateInMemoryDB(m.App)
		inMemoryDbProvider.Configure(&c)
	}

	loggingProvider, err := provider.GetLogging()

	if err != nil {
		return result, &c, err
	}

	err = loggingProvider.SetUpLogging(nn)

	if err != nil {
		return result, &c, err
	}

	loggingProvider.Configure(&c)

	return result, &c, nil
}

func (m *Maker) makeDependencies(c *config.AppConfig) (ctrl.Result, error) {
	depConfig := config.DependenciesConfig{}

	// Return if no deps

	deps := m.App.Spec.Dependencies

	if deps == nil || len(deps) == 0 {
		return ctrl.Result{}, nil
	}

	// Get all ClowdApps

	apps := crd.ClowdAppList{}
	err := m.Client.List(m.Ctx, &apps)

	if err != nil {
		return ctrl.Result{}, err
	}

	appMap := map[string]crd.ClowdApp{}

	for _, app := range apps.Items {
		if app.Spec.EnvName == m.App.Spec.EnvName {
			appMap[app.Name] = app
		}
	}

	// Iterate over all deps

	missingDeps := []string{}

	for _, dep := range m.App.Spec.Dependencies {
		depApp, exists := appMap[dep]

		if !exists {
			missingDeps = append(missingDeps, dep)
			continue
		}

		// If app has public endpoint, add it to app config

		svcName := types.NamespacedName{
			Name:      depApp.Name,
			Namespace: depApp.Namespace,
		}

		svc := core.Service{}
		err = m.Client.Get(m.Ctx, svcName, &svc)

		if err != nil {
			return ctrl.Result{}, err
		}

		for _, port := range svc.Spec.Ports {
			if port.Name == "web" {
				depConfig = append(depConfig, config.DependencyConfig{
					Hostname: fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace),
					Port:     int(port.Port),
					Name:     depApp.Name,
				})
			}
		}
	}

	if len(missingDeps) > 0 {
		// TODO: Emit event
		return ctrl.Result{Requeue: true}, nil
	}

	c.Dependencies = depConfig
	return ctrl.Result{}, nil
}
