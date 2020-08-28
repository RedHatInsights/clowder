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

package controllers

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cloudredhatcomv1alpha1 "cloud.redhat.com/whippoorwill/v2/api/v1alpha1"
	"cloud.redhat.com/whippoorwill/v2/controllers/config"
)

// InsightsAppReconciler reconciles a InsightsApp object
type InsightsAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *InsightsAppReconciler) makeService(req *ctrl.Request, iapp *cloudredhatcomv1alpha1.InsightsApp) error {

	labels := make(map[string]string)
	labels["app"] = iapp.ObjectMeta.Name

	ports := []core.ServicePort{}
	metricsPort := core.ServicePort{Name: "metrics", Port: 9000, Protocol: "TCP"}
	ports = append(ports, metricsPort)
	serviceName := iapp.ObjectMeta.Name

	if iapp.Spec.Web == true {
		webPort := core.ServicePort{Name: "web", Port: 8000, Protocol: "TCP"}
		ports = append(ports, webPort)
	}

	s := core.Service{}
	s.ObjectMeta = iapp.MakeObjectMeta()

	err := r.Client.Get(context.Background(), req.NamespacedName, &s)

	update := false

	if err == nil {
		update = true
	}

	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	s.Name = serviceName
	s.Spec.Selector = labels
	s.Spec.Ports = ports
	s.Namespace = req.Namespace

	if update {
		err = r.Client.Update(context.Background(), &s)
	} else {
		err = r.Client.Create(context.Background(), &s)
	}

	if err != nil {
		return err
	}
	return nil
}

func (r *InsightsAppReconciler) makeDeployment(iapp *cloudredhatcomv1alpha1.InsightsApp, d *apps.Deployment) {

	labels := map[string]string{
		"app": iapp.ObjectMeta.Name,
	}

	d.ObjectMeta = iapp.MakeObjectMeta()

	d.Spec.Replicas = iapp.Spec.MinReplicas
	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	d.Spec.Template.Spec.Volumes = iapp.Spec.Volumes
	d.Spec.Template.ObjectMeta.Labels = labels

	pullSecretRef := core.LocalObjectReference{Name: "quay-cloudservices-pull"}
	d.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{pullSecretRef}

	c := core.Container{
		Name:           iapp.ObjectMeta.Name,
		Image:          iapp.Spec.Image,
		Command:        iapp.Spec.Command,
		Args:           iapp.Spec.Args,
		Env:            iapp.Spec.Env,
		Resources:      iapp.Spec.Resources,
		LivenessProbe:  iapp.Spec.LivenessProbe,
		ReadinessProbe: iapp.Spec.ReadinessProbe,
		VolumeMounts:   iapp.Spec.VolumeMounts,
	}

	d.Spec.Template.Spec.Containers = []core.Container{c}
}

func (r *InsightsAppReconciler) persistConfig(req ctrl.Request, iapp cloudredhatcomv1alpha1.InsightsApp, c *config.AppConfig) error {

	ctx := context.Background()

	err := r.Client.Get(ctx, req.NamespacedName, &core.Secret{})

	update := (err == nil)

	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	jsonData, err := json.Marshal(c)

	if err != nil {
		return err
	}

	secret := core.Secret{
		ObjectMeta: iapp.MakeObjectMeta(),
		StringData: map[string]string{
			"cdappconfig.json": string(jsonData),
		},
	}

	if update {
		return r.Client.Update(ctx, &secret)
	}

	return r.Client.Create(ctx, &secret)
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsapps/status,verbs=get;update;patch

func updateOrErr(err error) (bool, error) {
	update := (err == nil)

	if err != nil && !k8serr.IsNotFound(err) {
		return false, err
	}

	return update, nil
}

// Reconcile function for InsightsAppReconciler
func (r *InsightsAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("insightsapp", req.NamespacedName)

	iapp := cloudredhatcomv1alpha1.InsightsApp{}
	err := r.Client.Get(ctx, req.NamespacedName, &iapp)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// TODO: requeue?
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	d := apps.Deployment{}
	err = r.Client.Get(ctx, req.NamespacedName, &d)

	update, err := updateOrErr(err)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.makeDeployment(&iapp, &d)

	c := config.New(8080, 9090, "/metrics", config.CloudWatch(
		config.CloudWatchConfig{
			AccessKeyID:     "mah_key",
			SecretAccessKey: "mah_sekret",
			Region:          "us-east-1",
			LogGroup:        iapp.ObjectMeta.Namespace,
		},
	))

	if err = r.persistConfig(req, iapp, c); err != nil {
		return ctrl.Result{}, err
	}

	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName: iapp.ObjectMeta.Name,
			},
		},
	})

	con := &d.Spec.Template.Spec.Containers[0]
	con.VolumeMounts = append(con.VolumeMounts, core.VolumeMount{
		Name:      "config-secret",
		MountPath: "/cdapp/",
	})

	if update {
		err = r.Client.Update(ctx, &d)
	} else {
		err = r.Client.Create(ctx, &d)
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.makeService(&req, &iapp)

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager for InsightsAppReconciler
func (r *InsightsAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudredhatcomv1alpha1.InsightsApp{}).
		Complete(r)
}
