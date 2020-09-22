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
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/makers"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=applications/status,verbs=get;update;patch

// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile fn
func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("application", req.NamespacedName)

	app := crd.Application{}
	err := r.Client.Get(ctx, req.NamespacedName, &app)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	env := crd.Environment{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: app.Namespace,
		Name:      app.Spec.Base,
	}, &env)

	if err != nil {
		return ctrl.Result{}, err
	}

	maker := makers.Maker{
		App:     &app,
		Env:     &env,
		Client:  r.Client,
		Ctx:     ctx,
		Request: &req,
		Log:     r.Log,
	}

	return maker.Make()
}

// SetupWithManager sets up wi
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("Setting up manager!")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.Application{}).
		Watches(
			&source.Kind{Type: &crd.Environment{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(r.appsToEnqueueUponEnvUpdate)},
		).
		Owns(&apps.Deployment{}).
		Owns(&core.Service{}).
		Complete(r)
}

func (r *ApplicationReconciler) appsToEnqueueUponEnvUpdate(a handler.MapObject) []reconcile.Request {
	reqs := []reconcile.Request{}
	ctx := context.Background()
	obj := types.NamespacedName{
		Name:      a.Meta.GetName(),
		Namespace: a.Meta.GetNamespace(),
	}

	// Get the Environment resource

	env := crd.Environment{}
	err := r.Client.Get(ctx, obj, &env)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return reqs
		}
		r.Log.Error(err, "Failed to fetch Environment")
		return nil
	}

	// Get all the Application resources

	appList := crd.ApplicationList{}
	r.Client.List(ctx, &appList)

	// Filter based on base attribute

	for _, app := range appList.Items {
		if app.Spec.Base == env.Name {
			// Add filtered resources to return result
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
		}
	}

	return reqs
}
