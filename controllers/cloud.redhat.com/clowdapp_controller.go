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
	"reflect"

	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta1"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	// Import the providers to initialize them
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"

	// These imports are to register the providers with the provider registration system
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/confighash"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/cronjob"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/database"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/dependencies"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/featureflags"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/inmemorydb"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/iqe"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/kafka"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/logging"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/metrics"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/objectstore"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/serviceaccount"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/servicemesh"
	_ "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/web"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
)

const appFinalizer = "finalizer.app.cloud.redhat.com"

// ClowdAppReconciler reconciles a ClowdApp object
type ClowdAppReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=serviceaccounts;configmaps;services;persistentvolumeclaims;secrets;events;namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;create;update;watch;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkaconnectors,verbs=get;list;watch
// +kubebuilder:rbac:groups=cyndi.cloud.redhat.com,resources=cyndipipelines,verbs=get;list;watch;create;update;patch;delete

// Reconcile fn
func (r *ClowdAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	qualifiedName := fmt.Sprintf("%s:%s", req.Namespace, req.Name)
	log := r.Log.WithValues("app", qualifiedName).WithValues("id", utils.RandString(5))
	ctx := context.WithValue(context.Background(), errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)
	app := crd.ClowdApp{}
	err := r.Client.Get(ctx, req.NamespacedName, &app)

	if app.Spec.Pods != nil {
		app.ConvertToNewShim()
	}

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	isAppMarkedForDeletion := app.GetDeletionTimestamp() != nil
	if isAppMarkedForDeletion {
		if contains(app.GetFinalizers(), appFinalizer) {
			if err := r.finalizeApp(log, &app); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&app, appFinalizer)
			err := r.Update(ctx, &app)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(app.GetFinalizers(), appFinalizer) {
		if err := r.addFinalizer(log, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("Reconciliation started", "app", fmt.Sprintf("%s:%s", app.Namespace, app.Name))

	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &app)

	env := crd.ClowdEnvironment{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name: app.Spec.EnvName,
	}, &env)

	if err != nil {
		r.Recorder.Eventf(&app, "Warning", "ClowdEnvMissing", "Clowder Environment [%s] is missing", app.Spec.EnvName)
		return ctrl.Result{Requeue: true}, err
	}

	if !env.IsReady() {
		r.Recorder.Eventf(&app, "Warning", "ClowdEnvNotReady", "Clowder Environment [%s] is not ready", app.Spec.EnvName)
		log.Info("Env not yet ready", "app", app.Name, "namespace", app.Namespace)
		return ctrl.Result{Requeue: true}, errors.New(fmt.Sprintf("Clowd Environment not ready: %s", env.Name))
	}

	cache := providers.NewObjectCache(ctx, r.Client, scheme)

	provider := providers.Provider{
		Client: r.Client,
		Ctx:    ctx,
		Env:    &env,
		Cache:  &cache,
	}

	var requeue = false

	provErr := r.runProviders(&provider, &app)

	if provErr != nil {
		if non_fatal := errors.HandleError(ctx, provErr); !non_fatal {
			return ctrl.Result{}, provErr
		} else {
			requeue = true
		}
	}

	cacheErr := cache.ApplyAll()

	if cacheErr != nil {
		if non_fatal := errors.HandleError(ctx, provErr); !non_fatal {
			return ctrl.Result{}, cacheErr
		} else {
			requeue = true
		}
	}

	if statusErr := SetDeploymentStatus(ctx, r.Client, &app); statusErr != nil {
		return ctrl.Result{}, err
	}

	app.Status.Ready = app.IsReady()

	err = r.Client.Status().Update(ctx, &app)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Delete all resources that are not used anymore
	if !requeue {
		r.Recorder.Eventf(&app, "Normal", "SuccessfulCreate", "created", app.GetClowdName())
		log.Info("Reconciliation successful", "app", fmt.Sprintf("%s:%s", app.Namespace, app.Name))
		err := cache.Reconcile(&app)
		if err != nil {
			return ctrl.Result{Requeue: requeue}, nil
		}
	} else {
		log.Info("Reconciliation partially successful", "app", fmt.Sprintf("%s:%s", app.Namespace, app.Name))
		r.Recorder.Eventf(&app, "Normal", "SuccessfulPartialCreate", "requeued", app.GetClowdName())
	}

	if err == nil {
		if _, ok := managedApps[app.GetIdent()]; !ok {
			managedApps[app.GetIdent()] = true
		}
		managedAppsMetric.Set(float64(len(managedApps)))
	}

	return ctrl.Result{Requeue: requeue}, nil
}

func ignoreStatusUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Always update if a deployment being watched changes - this allows
			// status rollups to occur
			if reflect.TypeOf(e.ObjectNew).String() == "*v1.Deployment" {
				return true
			}

			// Allow reconciliation if the env changed status
			if objOld, ok := e.ObjectOld.(*crd.ClowdEnvironment); ok {
				if objNew, ok := e.ObjectNew.(*crd.ClowdEnvironment); ok {
					if !objOld.Status.Ready && objNew.Status.Ready {
						return true
					}
				}
			}

			if objOld, ok := e.ObjectOld.(*strimzi.Kafka); ok {
				if objNew, ok := e.ObjectNew.(*strimzi.Kafka); ok {
					if (objOld.Status != nil && objNew.Status != nil) && len(objOld.Status.Listeners) != len(objNew.Status.Listeners) {
						return true
					}
				}
			}

			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}
}

// SetupWithManager sets up with Manager
func (r *ClowdAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("Setting up manager")
	utils.Log = r.Log.WithValues("name", "util")
	r.Recorder = mgr.GetEventRecorderFor("app")

	cache := mgr.GetCache()

	cache.IndexField(
		context.TODO(), &crd.ClowdApp{}, "spec.envName", func(o runtime.Object) []string {
			return []string{o.(*crd.ClowdApp).Spec.EnvName}
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdApp{}).
		Watches(
			&source.Kind{Type: &crd.ClowdEnvironment{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(r.appsToEnqueueUponEnvUpdate)},
		).
		Owns(&apps.Deployment{}).
		Owns(&core.Service{}).
		Owns(&core.ConfigMap{}).
		WithEventFilter(ignoreStatusUpdatePredicate()).
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

		if app.Spec.Pods != nil {
			app.ConvertToNewShim()
		}

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

func (r *ClowdAppReconciler) finalizeApp(reqLogger logr.Logger, a *crd.ClowdApp) error {

	delete(managedApps, a.GetIdent())

	managedAppsMetric.Set(float64(len(managedApps)))
	reqLogger.Info("Successfully finalized ClowdApp")
	return nil
}

func (r *ClowdAppReconciler) addFinalizer(reqLogger logr.Logger, a *crd.ClowdApp) error {
	reqLogger.Info("Adding Finalizer for the ClowdApp")
	controllerutil.AddFinalizer(a, appFinalizer)

	// Update CR
	err := r.Update(context.TODO(), a)
	if err != nil {
		reqLogger.Error(err, "Failed to update ClowdApp with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (r *ClowdAppReconciler) runProviders(provider *providers.Provider, a *crd.ClowdApp) error {

	c := config.AppConfig{}

	c.WebPort = utils.IntPtr(int(provider.Env.Spec.Providers.Web.Port))
	c.PublicPort = utils.IntPtr(int(provider.Env.Spec.Providers.Web.Port))
	privatePort := provider.Env.Spec.Providers.Web.PrivatePort
	if privatePort == 0 {
		privatePort = 10000
	}
	c.PrivatePort = utils.IntPtr(int(privatePort))
	c.MetricsPort = int(provider.Env.Spec.Providers.Metrics.Port)
	c.MetricsPath = provider.Env.Spec.Providers.Metrics.Path

	for _, provAcc := range providers.ProvidersRegistration.Registry {
		prov, err := provAcc.SetupProvider(provider)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("getprov: %s", provAcc.Name), err)
		}
		err = prov.Provide(a, &c)
		if err != nil {
			reterr := errors.Wrap(fmt.Sprintf("runapp: %s", provAcc.Name), err)
			reterr.Requeue = true
			return reterr
		}
	}

	return nil
}
