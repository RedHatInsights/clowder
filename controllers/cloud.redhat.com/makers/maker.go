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

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/database"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/inmemorydb"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/kafka"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/logging"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/objectstore"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	config []config.DependencyEndpoint
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
	Apply(obj runtime.Object) error
	Updater() utils.Updater
}

func (mu *makerUpdater) Apply(obj runtime.Object) error {
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
func (m *Maker) Make() error {
	c, err := m.runProviders()

	if err != nil {
		return err
	}

	if err := m.makeDependencies(c); err != nil {
		return err
	}

	hash, err := m.persistConfig(c)

	if err != nil {
		return err
	}

	for _, pod := range m.App.Spec.Pods {

		if err := m.makeDeployment(pod, hash); err != nil {
			return err
		}

		if err := m.makeService(pod); err != nil {
			return err
		}
	}
	return nil
}

func (m *Maker) makeService(pod crd.PodSpec) error {

	s := core.Service{}
	nn := types.NamespacedName{
		Name:      pod.Name,
		Namespace: m.Request.Namespace,
	}
	err := m.Client.Get(m.Ctx, nn, &s)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	ports := []core.ServicePort{
		{Name: "metrics", Port: m.Env.Spec.Metrics.Port, Protocol: "TCP"},
	}

	if pod.Web == true {
		webPort := core.ServicePort{Name: "web", Port: m.Env.Spec.Web.Port, Protocol: "TCP"}
		ports = append(ports, webPort)
	}

	utils.MakeService(&s, nn, map[string]string{"pod": nn.Name}, ports, m.App)

	return update.Apply(m.Ctx, m.Client, &s)
}

func (m *Maker) persistConfig(c *config.AppConfig) (string, error) {

	// In any case, we want to overwrite the secret, so this just
	// tests to see if the secret exists
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &core.Secret{})

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(c)
	if err != nil {
		return "", errors.Wrap("Failed to marshal config JSON", err)
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

	err = update.Apply(m.Ctx, m.Client, &secret)
	return hash, err
}

func (m *Maker) getConfig() (*config.AppConfig, error) {
	secret := core.Secret{}
	appConfig := config.AppConfig{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &secret)

	if err != nil {
		return nil, errors.Wrap("Failed to fetch config seret", err)
	}

	err = json.Unmarshal([]byte(secret.Data["cdappconfig.json"]), &appConfig)
	if err != nil {
		return nil, errors.Wrap("Failed to unmarshal JSON", err)
	}
	return &appConfig, nil
}

func applyPodAntiAffinity(t *core.PodTemplateSpec) {
	labelSelector := &metav1.LabelSelector{MatchLabels: t.Labels}
	t.Spec.Affinity = &core.Affinity{PodAntiAffinity: &core.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []core.WeightedPodAffinityTerm{{
			Weight: 100,
			PodAffinityTerm: core.PodAffinityTerm{
				LabelSelector: labelSelector,
				TopologyKey:   "failure-domain.beta.kubernetes.io/zone",
			},
		}, {
			Weight: 99,
			PodAffinityTerm: core.PodAffinityTerm{
				LabelSelector: labelSelector,
				TopologyKey:   "kubernetes.io/hostname",
			},
		}},
	}}
}

// This should probably take arguments for addtional volumes, so that we can
// add those and then do one Apply
func (m *Maker) makeDeployment(pod crd.PodSpec, hash string) error {

	nn := types.NamespacedName{
		Name:      pod.Name,
		Namespace: m.Request.Namespace,
	}

	d := apps.Deployment{}
	err := m.Client.Get(m.Ctx, nn, &d)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	initDeployment(m.App, m.Env, &d, nn, pod, hash)

	if err := update.Apply(m.Ctx, m.Client, &d); err != nil {
		return err
	}

	return nil
}

func initDeployment(app *crd.ClowdApp, env *crd.ClowdEnvironment, d *apps.Deployment, nn types.NamespacedName, pod crd.PodSpec, hash string) {
	labels := app.GetLabels()
	labels["pod"] = nn.Name
	fmt.Printf("initDeployment labels %v", labels)
	app.SetObjectMeta(d, crd.Name(pod.Name), crd.Labels(labels))

	d.Spec.Replicas = pod.MinReplicas
	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	d.Spec.Template.ObjectMeta.Labels = labels

	d.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
		{Name: "quay-cloudservices-pull"},
	}

	envvar := pod.Env
	envvar = append(envvar, core.EnvVar{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"})

	var livenessProbe core.Probe
	var readinessProbe core.Probe

	baseProbe := core.Probe{
		Handler: core.Handler{
			HTTPGet: &core.HTTPGetAction{
				Path:   "/healthz",
				Scheme: "HTTP",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: env.Spec.Web.Port,
				},
			},
		},
		FailureThreshold:    3,
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		TimeoutSeconds:      1,
	}
	if pod.LivenessProbe != nil {
		livenessProbe = *pod.LivenessProbe
	} else if pod.Web {
		livenessProbe = baseProbe
	}
	if pod.ReadinessProbe != nil {
		readinessProbe = *pod.ReadinessProbe
	} else {
		readinessProbe = baseProbe
		readinessProbe.InitialDelaySeconds = 45

	}

	var lcpu, lmemory, rcpu, rmemory resource.Quantity
	nullCPU := resource.Quantity{Format: resource.DecimalSI}
	nullMemory := resource.Quantity{Format: resource.BinarySI}

	if *pod.Resources.Limits.Cpu() != nullCPU {
		lcpu = pod.Resources.Limits["cpu"]
	} else {
		lcpu = env.Spec.ResourceDefaults.Limits["cpu"]
	}

	if *pod.Resources.Limits.Memory() != nullMemory {
		lmemory = pod.Resources.Limits["memory"]
	} else {
		lmemory = env.Spec.ResourceDefaults.Limits["memory"]
	}

	if *pod.Resources.Requests.Cpu() != nullCPU {
		rcpu = pod.Resources.Requests["cpu"]
	} else {
		rcpu = env.Spec.ResourceDefaults.Requests["cpu"]
	}

	if *pod.Resources.Requests.Memory() != nullMemory {
		rmemory = pod.Resources.Requests["memory"]
	} else {
		rmemory = env.Spec.ResourceDefaults.Requests["memory"]
	}

	resources := core.ResourceRequirements{
		Limits: core.ResourceList{
			"cpu":    lcpu,
			"memory": lmemory,
		},
		Requests: core.ResourceList{
			"cpu":    rcpu,
			"memory": rmemory,
		},
	}

	c := core.Container{
		Name:         nn.Name,
		Image:        pod.Image,
		Command:      pod.Command,
		Args:         pod.Args,
		Env:          envvar,
		Resources:    resources,
		VolumeMounts: pod.VolumeMounts,
		Ports: []core.ContainerPort{{
			Name:          "metrics",
			ContainerPort: env.Spec.Metrics.Port,
		}},
		ImagePullPolicy: core.PullIfNotPresent,
	}

	if (core.Probe{}) != livenessProbe {
		c.LivenessProbe = &livenessProbe
	}
	if (core.Probe{}) != readinessProbe {
		c.ReadinessProbe = &readinessProbe
	}

	if pod.Web {
		c.Ports = append(c.Ports, core.ContainerPort{
			Name:          "web",
			ContainerPort: env.Spec.Web.Port,
		})
	}

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "config-secret",
		MountPath: "/cdapp/",
	})

	d.Spec.Template.Spec.Containers = []core.Container{c}

	d.Spec.Template.Spec.Volumes = pod.Volumes
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName: app.ObjectMeta.Name,
			},
		},
	})

	applyPodAntiAffinity(&d.Spec.Template)

	annotations := make(map[string]string)
	annotations["configHash"] = hash
	d.Spec.Template.SetAnnotations(annotations)
}

func (m *Maker) runProviders() (*config.AppConfig, error) {
	provider := providers.Provider{
		Client: m.Client,
		Ctx:    m.Ctx,
		Env:    m.Env,
	}

	c := config.AppConfig{}

	c.WebPort = int(m.Env.Spec.Web.Port)
	c.MetricsPort = int(m.Env.Spec.Metrics.Port)
	c.MetricsPath = m.Env.Spec.Metrics.Path

	if err := objectstore.RunAppProvider(provider, &c, m.App); err != nil {
		return &c, errors.Wrap("setupenv: getobjectstore", err)
	}
	if err := logging.RunAppProvider(provider, &c, m.App); err != nil {
		return &c, errors.Wrap("setupenv: logging", err)
	}
	if err := kafka.RunAppProvider(provider, &c, m.App); err != nil {
		return &c, errors.Wrap("setupenv: kafka", err)
	}
	if err := inmemorydb.RunAppProvider(provider, &c, m.App); err != nil {
		return &c, errors.Wrap("setupenv: inmemorydb", err)
	}
	if err := database.RunAppProvider(provider, &c, m.App); err != nil {
		return &c, errors.Wrap("setupenv: database", err)
	}

	return &c, nil
}

func (m *Maker) makeDependencies(c *config.AppConfig) error {

	// Return if no deps

	deps := m.App.Spec.Dependencies

	if deps == nil || len(deps) == 0 {
		return nil
	}

	// Get all ClowdApps

	apps := crd.ClowdAppList{}
	err := m.Client.List(m.Ctx, &apps)

	if err != nil {
		return errors.Wrap("Failed to list apps", err)
	}

	// Iterate over all deps

	depConfig, missingDeps := makeDepConfig(m.Env.Spec.Web.Port, m.App, &apps)

	if len(missingDeps) > 0 {
		depVal := map[string][]string{"services": missingDeps}
		return &errors.MissingDependencies{MissingDeps: depVal}
	}

	c.Endpoints = depConfig
	return nil
}

func makeDepConfig(webPort int32, app *crd.ClowdApp, apps *crd.ClowdAppList) (depConfig []config.DependencyEndpoint, missingDeps []string) {
	appMap := map[string]crd.ClowdApp{}

	for _, app := range apps.Items {
		if app.Spec.EnvName == app.Spec.EnvName {
			appMap[app.Name] = app
		}
	}

	missingDeps = []string{}
	depConfig = []config.DependencyEndpoint{}

	for _, dep := range app.Spec.Dependencies {
		depApp, exists := appMap[dep]
		if !exists {
			missingDeps = append(missingDeps, dep)
			continue
		}

		// If app has public endpoint, add it to app config

		for _, pod := range depApp.Spec.Pods {
			if pod.Web {
				depConfig = append(depConfig, config.DependencyEndpoint{
					Hostname: fmt.Sprintf("%s.%s.svc", pod.Name, depApp.Namespace),
					Port:     int(webPort),
					Name:     pod.Name,
					App:      depApp.Name,
				})
			}
		}
	}

	return depConfig, missingDeps
}
