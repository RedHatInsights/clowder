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
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

func (r *InsightsAppReconciler) makeServices(req *ctrl.Request, iapp *cloudredhatcomv1alpha1.InsightsApp) error {
	labels := make(map[string]string)
	labels["app"] = iapp.ObjectMeta.Name

	if iapp.Spec.Port != nil {
		for _, containerPort := range iapp.Spec.Port {
			ctrl.Log.Info(fmt.Sprintf("%v", containerPort))
			serviceName := containerPort.Name
			if serviceName == "" {
				serviceName = fmt.Sprintf("%s-%s", iapp.ObjectMeta.Name, strconv.Itoa(int(containerPort.ContainerPort)))
			}
			namespacedName := types.NamespacedName{Name: serviceName, Namespace: req.NamespacedName.Namespace}
			s := core.Service{}

			err := r.Client.Get(context.Background(), namespacedName, &s)

			update := false

			if err == nil {
				update = true
			}

			if err != nil && !k8serr.IsNotFound(err) {
				return err
			}

			s.Name = serviceName
			port := core.ServicePort{Protocol: containerPort.Protocol, Port: containerPort.ContainerPort}
			s.Spec.Selector = labels
			s.Spec.Ports = []core.ServicePort{port}
			s.Namespace = req.Namespace

			if update {
				err = r.Client.Update(context.Background(), &s)
			} else {
				err = r.Client.Create(context.Background(), &s)
			}

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *InsightsAppReconciler) makeDeployment(iapp *cloudredhatcomv1alpha1.InsightsApp, d *apps.Deployment) {
	labels := make(map[string]string)
	labels["app"] = iapp.ObjectMeta.Name

	m := metav1.ObjectMeta{}
	m.Name = iapp.ObjectMeta.Name
	m.Namespace = iapp.ObjectMeta.Namespace
	m.Labels = labels

	owner := metav1.OwnerReference{}
	owner.APIVersion = iapp.APIVersion
	owner.Kind = iapp.Kind
	owner.Name = iapp.ObjectMeta.Name
	owner.UID = iapp.ObjectMeta.UID

	m.OwnerReferences = []metav1.OwnerReference{owner}

	d.ObjectMeta = m

	d.Spec.Replicas = iapp.Spec.MinReplicas
	selector := metav1.LabelSelector{}
	selector.MatchLabels = labels
	d.Spec.Selector = &selector
	d.Spec.Template.Spec.Volumes = iapp.Spec.Volumes
	d.Spec.Template.ObjectMeta.Labels = labels

	pullSecretRef := core.LocalObjectReference{}
	pullSecretRef.Name = "quay-cloudservices-pull"
	d.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{pullSecretRef}

	c := core.Container{}
	c.Name = iapp.ObjectMeta.Name
	c.Image = iapp.Spec.Image
	c.Command = iapp.Spec.Command
	c.Args = iapp.Spec.Args
	c.Env = iapp.Spec.Env
	c.Resources = iapp.Spec.Resources
	c.LivenessProbe = iapp.Spec.LivenessProbe
	c.ReadinessProbe = iapp.Spec.ReadinessProbe
	c.VolumeMounts = iapp.Spec.VolumeMounts

	d.Spec.Template.Spec.Containers = []core.Container{c}
}

func (r *InsightsAppReconciler) persistConfig(req ctrl.Request, iapp cloudredhatcomv1alpha1.InsightsApp, c *config.AppConfig) error {

	err := r.Client.Get(context.Background(), req.NamespacedName, &core.Secret{})

	update := (err == nil)

	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	jsonData, err := json.Marshal(c)

	if err != nil {
		return err
	}

	owner := metav1.OwnerReference{}
	owner.APIVersion = iapp.APIVersion
	owner.Kind = iapp.Kind
	owner.Name = iapp.ObjectMeta.Name
	owner.UID = iapp.ObjectMeta.UID

	secret := core.Secret{}
	secret.ObjectMeta = metav1.ObjectMeta{
		Name:            iapp.ObjectMeta.Name,
		Namespace:       iapp.ObjectMeta.Namespace,
		OwnerReferences: []metav1.OwnerReference{owner},
	}
	secret.StringData = map[string]string{
		"cdappconfig.json": string(jsonData),
	}

	if update {
		secret.Data = nil
		return r.Client.Update(context.Background(), &secret)
	}

	return r.Client.Create(context.Background(), &secret)
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsapps/status,verbs=get;update;patch

func (r *InsightsAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("insightsapp", req.NamespacedName)

	iapp := cloudredhatcomv1alpha1.InsightsApp{}
	err := r.Client.Get(context.Background(), req.NamespacedName, &iapp)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// TODO: requeue?
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	d := apps.Deployment{}
	err = r.Client.Get(context.Background(), req.NamespacedName, &d)

	update := false

	if err == nil {
		update = true
	}

	if err != nil && !k8serr.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	r.makeDeployment(&iapp, &d)

	cb := config.NewBuilder()
	cb.CloudWatch(&config.CloudWatchConfig{
		AccessKeyID:     "mah_key",
		SecretAccessKey: "mah_sekret",
		Region:          "us-east-1",
		LogGroup:        iapp.ObjectMeta.Namespace,
	})
	c := cb.Build()
	c.WebPort = 8080
	c.MetricsPort = 9090
	c.MetricsPath = "/metrics"

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
		err = r.Client.Update(context.Background(), &d)
	} else {
		err = r.Client.Create(context.Background(), &d)
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.makeServices(&req, &iapp)

	if err != nil {
		return ctrl.Result{}, err
	}

	// For each app that's new:
	//   Generate ConfigMap
	//     kafka bootstrap env
	//     metric port
	//     consumer group
	//     topic
	//   Create Deployment with:
	//     config mount
	//     pod anti-affinity
	//     resource limits
	//     limited service account
	//     image spec
	//	   pull secrets
	//     metric port
	//     web port
	//   Create ServiceMonitor
	//   Create PrometheusRules for lag

	return ctrl.Result{}, nil
}

func (r *InsightsAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudredhatcomv1alpha1.InsightsApp{}).
		Complete(r)
}
