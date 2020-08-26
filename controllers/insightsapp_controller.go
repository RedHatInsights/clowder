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

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cloudredhatcomv1alpha1 "cloud.redhat.com/whippoorwill/v2/api/v1alpha1"
)

// InsightsAppReconciler reconciles a InsightsApp object
type InsightsAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsapps/status,verbs=get;update;patch

func (r *InsightsAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("insightsapp", req.NamespacedName)

	iapp := &cloudredhatcomv1alpha1.InsightsApp{}
	err := r.Client.Get(context.Background(), req.NamespacedName, iapp)

	if err != nil {
		return ctrl.Result{}, err
	}

	labels := make(map[string]string)
	labels["app"] = iapp.ObjectMeta.Name

	m := metav1.ObjectMeta{}
	m.Name = iapp.ObjectMeta.Name
	m.Namespace = req.NamespacedName.Namespace
	m.Labels = labels

	d := apps.Deployment{}
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

	err = r.Client.Create(context.Background(), &d)

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
	//   Create Service
	//   Create ServiceMonitor
	//   Create PrometheusRules for lag

	return ctrl.Result{}, nil
}

func (r *InsightsAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudredhatcomv1alpha1.InsightsApp{}).
		Complete(r)
}
