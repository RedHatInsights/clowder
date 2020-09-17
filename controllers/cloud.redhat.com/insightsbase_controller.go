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
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/makers"
)

// InsightsBaseReconciler reconciles a InsightsBase object
type InsightsBaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsbases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsbases/status,verbs=get;update;patch

//Reconcile fn
func (r *InsightsBaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("insightsbase", req.NamespacedName)

	base := crd.InsightsBase{}
	err := r.Client.Get(ctx, req.NamespacedName, &base)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	maker := makers.Maker{
		Ctx:     ctx,
		Client:  r.Client,
		Base:    &base,
		Request: &req,
		Log:     r.Log,
	}

	if base.Spec.ObjectStore.Provider == "minio" {
		err = makers.MakeMinio(r.Client, ctx, req, &base)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if base.Spec.Kafka.Provider == "local" {
		err = makers.MakeLocalZookeeper(&maker)

		if err != nil {
			return ctrl.Result{}, err
		}

		err = makers.MakeLocalKafka(&maker)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up with manager
func (r *InsightsBaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.InsightsBase{}).
		Complete(r)
}
