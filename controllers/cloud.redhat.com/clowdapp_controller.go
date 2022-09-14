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
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	// Import the providers to initialize them

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	// These imports are to register the providers with the provider registration system
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/autoscaler"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/confighash"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/cronjob"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/database"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/dependencies"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/featureflags"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/headless"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/inmemorydb"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/iqe"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/kafka"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/logging"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/metrics"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/namespace"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/objectstore"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/pullsecrets"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/serviceaccount"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/servicemesh"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sidecar"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/web"

	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

const appFinalizer = "finalizer.app.cloud.redhat.com"

type ReconciliationMetrics struct {
	appName            string
	envName            string
	reconcileStartTime time.Time
	metricsEnabled     bool
}

func (rm *ReconciliationMetrics) init(clowdAppName string, clowdEnvName string) {
	rm.appName = clowdAppName
	rm.envName = clowdEnvName
	rm.metricsEnabled = clowderconfig.LoadedConfig.Features.ReconciliationMetrics
}

func (rm *ReconciliationMetrics) start() {
	if !rm.metricsEnabled {
		return
	}
	rm.reconcileStartTime = time.Now()
}

func (rm *ReconciliationMetrics) stop() {
	if !rm.metricsEnabled {
		return
	}
	elapsedTime := time.Since(rm.reconcileStartTime).Seconds()
	reconciliationMetrics.With(prometheus.Labels{"app": rm.appName, "env": rm.envName}).Observe(elapsedTime)
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=serviceaccounts;configmaps;services;persistentvolumeclaims;secrets;events;namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;create;update;watch;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkausers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keda.sh,resources=scaledobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cyndi.cloud.redhat.com,resources=cyndipipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheuses;servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkaconnectors,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=endpoints;pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses;networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=ingresses,verbs=get;list

// ClowdAppReconciler reconciles a ClowdApp object
type ClowdAppReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile fn
func (r *ClowdAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("app", req.Name).WithValues("rid", utils.RandString(5)).WithValues("namespace", req.Namespace)
	ctx = context.WithValue(ctx, errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)
	app := crd.ClowdApp{}

	if getAppErr := r.Client.Get(ctx, req.NamespacedName, &app); getAppErr != nil {
		if k8serr.IsNotFound(getAppErr) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		log.Info("App not found", "env", app.Spec.EnvName, "app", app.GetIdent(), "err", getAppErr)
		return ctrl.Result{}, getAppErr
	}

	if _, ok := presentApps[app.GetIdent()]; !ok {
		presentApps[app.GetIdent()] = true
	}
	presentAppsMetric.Set(float64(len(presentApps)))

	delete(managedApps, app.GetIdent())

	defer func() {
		managedAppsMetric.Set(float64(len(managedApps)))
	}()

	log = log.WithValues("env", app.Spec.EnvName)

	isAppMarkedForDeletion := app.GetDeletionTimestamp() != nil
	if isAppMarkedForDeletion {
		if contains(app.GetFinalizers(), appFinalizer) {
			if finalizeErr := r.finalizeApp(log, &app); finalizeErr != nil {
				log.Info("Cloud not finalize", "err", finalizeErr)
				return ctrl.Result{}, finalizeErr
			}

			controllerutil.RemoveFinalizer(&app, appFinalizer)
			removeFinalizeErr := r.Update(ctx, &app)
			if removeFinalizeErr != nil {
				log.Info("Cloud not remove finalizer", "err", removeFinalizeErr)
				return ctrl.Result{}, removeFinalizeErr
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(app.GetFinalizers(), appFinalizer) {
		if addFinalizeErr := r.addFinalizer(log, &app); addFinalizeErr != nil {
			log.Info("Cloud not add finalizer", "err", addFinalizeErr)
			return ctrl.Result{}, addFinalizeErr
		}
	}

	if ReadEnv() == app.Spec.EnvName {
		r.Recorder.Eventf(&app, "Warning", "ClowdEnvLocked", "Clowder Environment [%s] is locked", app.Spec.EnvName)
		log.Info("Env currently being reconciled")
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Reconciliation started")
	reconciliationMetrics := ReconciliationMetrics{}
	reconciliationMetrics.init(app.Name, app.Spec.EnvName)
	reconciliationMetrics.start()

	if clowderconfig.LoadedConfig.Features.PerProviderMetrics {
		requestMetrics.With(prometheus.Labels{"type": "app", "name": app.GetIdent()}).Inc()
	}

	if app.Spec.Disabled {
		log.Info("Reconciliation aborted - set to be disabled")
		return ctrl.Result{}, nil
	}

	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &app)

	env := crd.ClowdEnvironment{}

	if getEnvErr := r.Client.Get(ctx, types.NamespacedName{Name: app.Spec.EnvName}, &env); getEnvErr != nil {
		log.Info("ClowdEnv missing", "err", getEnvErr)
		r.Recorder.Eventf(&app, "Warning", "ClowdEnvMissing", "Clowder Environment [%s] is missing", app.Spec.EnvName)
		if setClowdStatusErr := SetClowdAppConditions(ctx, r.Client, &app, crd.ReconciliationFailed, getEnvErr); setClowdStatusErr != nil {
			log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{}, getEnvErr
	}

	ens := &core.Namespace{}
	if getNSErr := r.Client.Get(ctx, types.NamespacedName{Name: env.Status.TargetNamespace}, ens); getNSErr != nil {
		return ctrl.Result{Requeue: true}, getNSErr
	}

	if ens.ObjectMeta.DeletionTimestamp != nil {
		log.Info("Env target namespace is to be deleted - skipping reconcile")
		return ctrl.Result{}, nil
	}

	ans := &core.Namespace{}
	if getAppNSErr := r.Client.Get(ctx, types.NamespacedName{Name: app.Namespace}, ans); getAppNSErr != nil {
		return ctrl.Result{Requeue: true}, getAppNSErr
	}

	if ans.ObjectMeta.DeletionTimestamp != nil {
		log.Info("App namespace is to be deleted - skipping reconcile")
		return ctrl.Result{}, nil
	}

	if env.Generation != env.Status.Generation {
		r.Recorder.Eventf(&app, "Warning", "ClowdEnvNotReconciled", "Clowder Environment [%s] is not reconciled", app.Spec.EnvName)
		log.Info("Env not yet reconciled")
		if setClowdStatusErr := SetClowdAppConditions(ctx, r.Client, &app, crd.ReconciliationFailed, fmt.Errorf("clowd env not reconciled")); setClowdStatusErr != nil {
			log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{}, nil
	}

	if !env.IsReady() {
		r.Recorder.Eventf(&app, "Warning", "ClowdEnvNotReady", "Clowder Environment [%s] is not ready", app.Spec.EnvName)
		log.Info("Env not yet ready")
		if setClowdStatusErr := SetClowdAppConditions(ctx, r.Client, &app, crd.ReconciliationFailed, fmt.Errorf("clowd env not ready")); setClowdStatusErr != nil {
			log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}

		return ctrl.Result{Requeue: true}, nil
	}

	cacheConfig := rc.NewCacheConfig(Scheme, errors.ClowdKey("log"), ProtectedGVKs, DebugOptions)

	cache := rc.NewObjectCache(ctx, r.Client, cacheConfig)
	//cache := providers.NewObjectCache(ctx, r.Client, scheme)

	provider := providers.Provider{
		Client: r.Client,
		Ctx:    ctx,
		Env:    &env,
		Cache:  &cache,
		Log:    log,
	}

	if provErr := r.runProviders(log, &provider, &app); provErr != nil {
		r.Recorder.Eventf(&app, "Warning", "FailedReconciliation", "Clowdapp requeued [%s]", app.GetClowdName())
		if setClowdStatusErr := SetClowdAppConditions(ctx, r.Client, &app, crd.ReconciliationFailed, provErr); setClowdStatusErr != nil {
			log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		log.Info("Provider error", "err", provErr)
		return ctrl.Result{Requeue: true}, provErr
	}

	cacheErr := cache.ApplyAll()

	if cacheErr != nil {
		r.Recorder.Eventf(&app, "Warning", "FailedReconciliation", "Clowdapp requeued [%s]", app.GetClowdName())
		if setClowdStatusErr := SetClowdAppConditions(ctx, r.Client, &app, crd.ReconciliationFailed, cacheErr); setClowdStatusErr != nil {
			log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		log.Info("Cache error", "err", cacheErr)
		return ctrl.Result{Requeue: true}, cacheErr
	}

	if statusErr := SetAppResourceStatus(ctx, r.Client, &app); statusErr != nil {
		log.Info("Set status error", "err", statusErr)
		return ctrl.Result{Requeue: true}, statusErr
	}

	opts := []client.ListOption{
		client.MatchingLabels{app.GetPrimaryLabel(): app.GetClowdName()},
		client.InNamespace(app.Namespace),
	}

	// Delete all resources that are not used anymore
	rErr := cache.Reconcile(app.GetUID(), opts...)
	if rErr != nil {
		log.Info("Reconcile error", "err", rErr)
		return ctrl.Result{Requeue: true}, nil
	}

	if setClowdStatusErr := SetClowdAppConditions(ctx, r.Client, &app, crd.ReconciliationSuccessful, nil); setClowdStatusErr != nil {
		log.Info("Set status error", "err", setClowdStatusErr)
		return ctrl.Result{Requeue: true}, setClowdStatusErr
	}

	managedApps[app.GetIdent()] = true

	r.Recorder.Eventf(&app, "Normal", "SuccessfulReconciliation", "Clowdapp reconciled [%s]", app.GetClowdName())
	log.Info("Reconciliation successful")

	reconciliationMetrics.stop()
	return ctrl.Result{}, nil
}

// SetupWithManager sets up with Manager
func (r *ClowdAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("Setting up manager")
	utils.Log = r.Log.WithValues("name", "util")
	r.Recorder = mgr.GetEventRecorderFor("app")

	cache := mgr.GetCache()

	cache.IndexField(
		context.TODO(), &crd.ClowdApp{}, "spec.envName", func(o client.Object) []string {
			return []string{o.(*crd.ClowdApp).Spec.EnvName}
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdApp{}).
		Watches(
			&source.Kind{Type: &crd.ClowdEnvironment{}},
			handler.EnqueueRequestsFromMapFunc(r.appsToEnqueueUponEnvUpdate),
			builder.WithPredicates(environmentPredicate(r.Log, "app")),
		).
		Owns(&apps.Deployment{}, builder.WithPredicates(deploymentPredicate(r.Log, "app"))).
		Owns(&core.Service{}, builder.WithPredicates(generationOnlyPredicate(r.Log, "app"))).
		Owns(&core.ConfigMap{}, builder.WithPredicates(generationOnlyPredicate(r.Log, "app"))).
		Owns(&core.Secret{}, builder.WithPredicates(alwaysPredicate(r.Log, "app"))).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Duration(500*time.Millisecond), time.Duration(60*time.Second)),
		}).
		Complete(r)
}

func (r *ClowdAppReconciler) appsToEnqueueUponEnvUpdate(a client.Object) []reconcile.Request {
	reqs := []reconcile.Request{}
	ctx := context.Background()
	obj := types.NamespacedName{
		Name:      a.GetName(),
		Namespace: a.GetNamespace(),
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

	appList, err := env.GetAppsInEnv(ctx, r.Client)
	if err != nil {
		r.Log.Error(err, "Failed to fetch ClowdApps")
		return nil
	}

	// Filter based on base attribute

	for _, app := range appList.Items {
		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		})
	}

	return reqs
}

func (r *ClowdAppReconciler) finalizeApp(reqLogger logr.Logger, a *crd.ClowdApp) error {

	delete(managedApps, a.GetIdent())
	managedAppsMetric.Set(float64(len(managedApps)))

	delete(presentApps, a.GetIdent())
	presentAppsMetric.Set(float64(len(presentApps)))

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

func updateMetadata(app *crd.ClowdApp, appConfig *config.AppConfig) {
	metadata := config.AppMetadata{}

	for _, deployment := range app.Spec.Deployments {
		deploymentMetadata := config.DeploymentMetadata{
			Name:  deployment.Name,
			Image: deployment.PodSpec.Image,
		}
		metadata.Deployments = append(metadata.Deployments, deploymentMetadata)
	}

	appConfig.Metadata = &metadata
	appConfig.Metadata.Name = &app.Name
	appConfig.Metadata.EnvName = &app.Spec.EnvName
}

func (r *ClowdAppReconciler) runProviders(log logr.Logger, provider *providers.Provider, a *crd.ClowdApp) error {

	c := config.AppConfig{}

	// Update app metadata
	updateMetadata(a, &c)

	for _, provAcc := range providers.ProvidersRegistration.Registry {
		provutils.DebugLog(log, "running provider:", "name", provAcc.Name, "order", provAcc.Order)
		start := time.Now()
		prov, err := provAcc.SetupProvider(provider)
		elapsed := time.Since(start).Seconds()
		providerMetrics.With(prometheus.Labels{"provider": provAcc.Name, "source": "clowdenv"}).Observe(elapsed)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("getprov: %s", provAcc.Name), err)
		}
		start = time.Now()
		err = prov.Provide(a, &c)
		elapsed = time.Since(start).Seconds()
		providerMetrics.With(prometheus.Labels{"provider": provAcc.Name, "source": "clowdapp"}).Observe(elapsed)
		if err != nil {
			reterr := errors.Wrap(fmt.Sprintf("runapp: %s", provAcc.Name), err)
			reterr.Requeue = true
			return reterr
		}
		provutils.DebugLog(log, "running provider: complete", "name", provAcc.Name, "order", provAcc.Order, "elapsed", fmt.Sprintf("%f", elapsed))
	}

	return nil
}
