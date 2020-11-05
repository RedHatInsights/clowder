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
	"fmt"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/makers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
)

// ClowdAppReconciler reconciles a ClowdApp object
type ClowdAppReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;secrets;events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkas,verbs=get;list;watch

// Reconcile fn
func (r *ClowdAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	qualifiedName := fmt.Sprintf("%s:%s", req.Namespace, req.Name)
	log := r.Log.WithValues("app", qualifiedName)
	ctx := context.WithValue(context.Background(), errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)

	app := crd.ClowdApp{}
	err := r.Client.Get(ctx, req.NamespacedName, &app)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &app)

	env := crd.ClowdEnvironment{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name: app.Spec.EnvName,
	}, &env)

	if err != nil {
		r.Recorder.Eventf(&app, "warning", "ClowdEnvMissing", "Clowder Environment [%s] is missing", app.Spec.EnvName)
		return ctrl.Result{}, err
	}

	maker, err := makers.New(&makers.Maker{
		App:     &app,
		Env:     &env,
		Client:  r.Client,
		Ctx:     ctx,
		Request: &req,
		Log:     r.Log,
	})

	err = maker.Make()

	if err == nil {
		r.Log.Info("Reconciliation successful", "app", app.Name)
	}

	requeue := errors.HandleError(ctx, err)
	return ctrl.Result{Requeue: requeue}, nil
}

// SetupWithManager sets up wi
func (r *ClowdAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("Setting up manager")
	utils.Log = r.Log.WithValues("name", "util")
	r.Recorder = mgr.GetEventRecorderFor("app")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdApp{}).
		Watches(
			&source.Kind{Type: &crd.ClowdEnvironment{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(r.appsToEnqueueUponEnvUpdate)},
		).
		Owns(&apps.Deployment{}).
		Owns(&core.Service{}).
		Complete(r)
}

func (r *ClowdAppReconciler) appsToEnqueueUponEnvUpdate(a handler.MapObject) []reconcile.Request {
	reqs := []reconcile.Request{}
	ctx := context.Background()
	obj := types.NamespacedName{
		Name:      a.Meta.GetName(),
		Namespace: a.Meta.GetNamespace(),
	}

	// Get the ClowdEnvironment resource

	env := crd.ClowdEnvironment{}
	err := r.Client.Get(ctx, obj, &env)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return reqs
		}
		r.Log.Error(err, "Failed to fetch ClowdEnvironment")
		return nil
	}

	// Get all the ClowdApp resources

	appList := crd.ClowdAppList{}
	r.Client.List(ctx, &appList)

	// Filter based on base attribute

	for _, app := range appList.Items {
		if app.Spec.EnvName == env.Name {
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
