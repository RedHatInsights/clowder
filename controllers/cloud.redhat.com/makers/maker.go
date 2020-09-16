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
	b64 "encoding/base64"
	"encoding/json"
	"fmt"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func b64decode(s *core.Secret, key string) (string, error) {
	decoded, err := b64.StdEncoding.DecodeString(string(s.Data[key]))

	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

//SubMaker interface defines interface for making sub objects
type SubMaker interface {
	Make() (ctrl.Result, error)
	ApplyConfig(c *config.AppConfig)
}

//Maker struct for passing variables into SubMakers
type Maker struct {
	App     *crd.InsightsApp
	Base    *crd.InsightsBase
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

func (m *Maker) getSubMakers() []SubMaker {
	return []SubMaker{
		&DependencyMaker{Maker: m},
		&KafkaMaker{Maker: m},
		&DatabaseMaker{Maker: m},
		&LoggingMaker{Maker: m},
		&ObjectStoreMaker{Maker: m},
		&InMemoryDBMaker{Maker: m},
	}
}

//Make generates objects and dependencies for operator
func (m *Maker) Make() (ctrl.Result, error) {
	configs := []config.ConfigOption{}

	for _, sm := range m.getSubMakers() {
		result, err := sm.Make()

		if err != nil || result.Requeue || result.RequeueAfter.Seconds() > 0.0 {
			return result, err
		}

		configs = append(configs, sm.ApplyConfig)
	}

	configs = append(configs, config.Web(int(m.Base.Spec.Web.Port)))
	configs = append(configs, config.Metrics(m.Base.Spec.Metrics.Path, int(m.Base.Spec.Metrics.Port)))

	c := config.New(configs...)

	hash, err := m.persistConfig(c)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := m.makeDeployment(hash); err != nil {
		return ctrl.Result{}, err
	}

	if err := m.makeService(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (m *Maker) makeService() error {

	s := core.Service{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &s)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	ports := []core.ServicePort{
		{Name: "metrics", Port: m.Base.Spec.Metrics.Port, Protocol: "TCP"},
	}

	if m.App.Spec.Web == true {
		webPort := core.ServicePort{Name: "web", Port: m.Base.Spec.Web.Port, Protocol: "TCP"}
		ports = append(ports, webPort)
	}

	m.App.SetObjectMeta(&s)
	s.Spec.Selector = m.App.GetLabels()
	s.Spec.Ports = ports

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
		return "", err
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

	return hash, update.Apply(m.Ctx, m.Client, &secret)
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
func (m *Maker) makeDeployment(hash string) error {

	d := apps.Deployment{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &d)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return err
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
				Path:   "/api/ingress/ping",
				Scheme: "HTTP",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8000,
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
			ContainerPort: m.Base.Spec.Metrics.Port,
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
			ContainerPort: m.Base.Spec.Web.Port,
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
	if err = update.Apply(m.Ctx, m.Client, &d); err != nil {
		return err
	}

	return nil
}
