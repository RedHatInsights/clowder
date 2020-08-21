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

	// Fetch InsightsApps
	app := &cloudredhatcomv1alpha1.InsightsApp{}
	r.Client.Get(context.Background(), req.NamespacedName, app)
	r.Log.Info("app: " + app.ObjectMeta.Name)
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
