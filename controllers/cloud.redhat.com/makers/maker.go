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
	b64 "encoding/base64"
	"encoding/json"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updater bool

func (u *updater) Apply(ctx context.Context, cl client.Client, obj runtime.Object) error {
	if *u {
		return cl.Update(ctx, obj)
	}
	return cl.Create(ctx, obj)
}

func updateOrErr(err error) (updater, error) {
	update := updater(err == nil)

	if err != nil && !k8serr.IsNotFound(err) {
		return update, err
	}

	return update, nil
}

func b64decode(s *core.Secret, key string) (string, error) {
	decoded, err := b64.StdEncoding.DecodeString(string(s.Data[key]))

	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

//SubMaker interface defines interface for making sub objects
type SubMaker interface {
	Make() error
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

func (m *Maker) getSubMakers() []SubMaker {
	return []SubMaker{
		&KafkaMaker{Maker: m},
		&DatabaseMaker{Maker: m},
		&LoggingMaker{Maker: m},
		&ObjectStoreMaker{Maker: m},
	}
}

//Make generates objects and dependencies for operator
func (m *Maker) Make() error {
	configs := []config.ConfigOption{}

	for _, sm := range m.getSubMakers() {
		err := sm.Make()

		if err != nil {
			return err
		}

		configs = append(configs, sm.ApplyConfig)
	}

	configs = append(configs, config.Web(m.Base.Spec.Web.Port))
	configs = append(configs, config.Metrics(m.Base.Spec.Metrics.Path, m.Base.Spec.Metrics.Port))

	c := config.New(configs...)

	if err := m.persistConfig(c); err != nil {
		return err
	}

	if err := m.makeDeployment(); err != nil {
		return err
	}

	if err := m.makeService(); err != nil {
		return err
	}

	return nil
}

func (m *Maker) makeService() error {

	s := core.Service{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &s)

	update, err := updateOrErr(err)
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

func (m *Maker) persistConfig(c *config.AppConfig) error {

	// In any case, we want to overwrite the secret, so this just
	// tests to see if the secret exists
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &core.Secret{})

	update, err := updateOrErr(err)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(c)
	if err != nil {
		return err
	}

	secret := core.Secret{
		StringData: map[string]string{
			"cdappconfig.json": string(jsonData),
		},
	}

	m.App.SetObjectMeta(&secret)

	return update.Apply(m.Ctx, m.Client, &secret)
}

// This should probably take arguments for addtional volumes, so that we can add those and then do one Apply
func (m *Maker) makeDeployment() error {

	d := apps.Deployment{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &d)

	update, err := updateOrErr(err)
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

	c := core.Container{
		Name:           m.App.ObjectMeta.Name,
		Image:          m.App.Spec.Image,
		Command:        m.App.Spec.Command,
		Args:           m.App.Spec.Args,
		Env:            m.App.Spec.Env,
		Resources:      m.App.Spec.Resources,
		LivenessProbe:  m.App.Spec.LivenessProbe,
		ReadinessProbe: m.App.Spec.ReadinessProbe,
		VolumeMounts:   m.App.Spec.VolumeMounts,
		Ports: []core.ContainerPort{{
			Name:          "metrics",
			ContainerPort: m.Base.Spec.Metrics.Port,
		}},
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

	if err = update.Apply(m.Ctx, m.Client, &d); err != nil {
		return err
	}

	return nil
}
