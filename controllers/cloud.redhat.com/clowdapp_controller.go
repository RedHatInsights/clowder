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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	// Import the providers to initialize them

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"

	// These imports are to register the providers with the provider registration system
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/autoscaler"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/confighash"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/cronjob"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/database"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/dependencies"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/featureflags"
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

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"

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
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapprefs,verbs=get;list;watch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapprefs/status,verbs=get;update;patch
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
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates;issuers,verbs=get;list;create;update;patch;delete

// ClowdAppReconciler reconciles a ClowdApp object
type ClowdAppReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	HashCache *hashcache.HashCache
}

type Watcher struct {
	obj    client.Object
	filter HandlerFuncBuilder
}

// Reconcile fn
func (r *ClowdAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("app", req.Name).WithValues("rid", utils.RandString(5)).WithValues("namespace", req.Namespace)
	ctx = context.WithValue(ctx, errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)
	app := crd.ClowdApp{}

	defer func() {
		managedAppsMetric.Set(float64(len(managedApps)))
	}()

	log.Info("Reconciliation started")

	reconciliation := ClowdAppReconciliation{
		ctx:       ctx,
		client:    r.Client,
		recorder:  r.Recorder,
		app:       &app,
		log:       &log,
		req:       &req,
		config:    &config.AppConfig{},
		hashCache: r.HashCache,
	}
	res, err := reconciliation.Reconcile()
	if err != nil {
		if shouldSkipReconciliation(err) {
			log.Info("skipping", "error", err.Error(), "skipping", "true", "requeue", res.Requeue)
			return res, nil
		}
		log.Error(err, "error in reconciliation", "skipping", "false", "requeue", res.Requeue)
		return res, err
	}

	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciler) setupWatch(ctrlr *builder.Builder, mgr ctrl.Manager, obj client.Object, handlerBuilder HandlerFuncBuilder) error {
	handler, err := createNewHandler(mgr, r.Scheme, handlerBuilder, r.Log, "app", &crd.ClowdApp{}, r.HashCache)
	if err != nil {
		return err
	}
	ctrlr.Watches(obj, handler)

	return nil
}

// SetupWithManager sets up with Manager
func (r *ClowdAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("Setting up manager")
	utils.Log = r.Log.WithValues("name", "util")
	r.Recorder = mgr.GetEventRecorderFor("app")

	cache := mgr.GetCache()

	if err := cache.IndexField(
		context.TODO(), &crd.ClowdApp{}, "spec.envName", func(o client.Object) []string {
			return []string{o.(*crd.ClowdApp).Spec.EnvName}
		}); err != nil {
		return err
	}

	ctrlr := ctrl.NewControllerManagedBy(mgr).For(&crd.ClowdApp{})
	ctrlr.Watches(
		&crd.ClowdEnvironment{},
		handler.EnqueueRequestsFromMapFunc(r.appsToEnqueueUponEnvUpdate),
		builder.WithPredicates(environmentPredicate(r.Log, "app")),
	)

	ctrlr.Watches(
		&crd.ClowdAppRef{},
		handler.EnqueueRequestsFromMapFunc(r.appsToEnqueueUponAppRefUpdate),
		builder.WithPredicates(predicate.GenerationChangedPredicate{}),
	)

	watchers := []Watcher{
		{obj: &apps.Deployment{}, filter: deploymentFilter},
		{obj: &core.Service{}, filter: generationOnlyFilter},
		{obj: &core.ConfigMap{}, filter: generationOnlyFilter},
		{obj: &core.Secret{}, filter: alwaysFilter},
	}

	for _, watcher := range watchers {
		err := r.setupWatch(ctrlr, mgr, watcher.obj, watcher.filter)
		if err != nil {
			return err
		}
	}

	ctrlr.WithOptions(controller.Options{
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Duration(500*time.Millisecond), time.Duration(60*time.Second)),
	})
	return ctrlr.Complete(r)
}

func (r *ClowdAppReconciler) appsToEnqueueUponEnvUpdate(ctx context.Context, a client.Object) []reconcile.Request {
	reqs := []reconcile.Request{}
	obj := types.NamespacedName{
		Name:      a.GetName(),
		Namespace: a.GetNamespace(),
	}

	// Get the ClowdEnvironment resource

	env := crd.ClowdEnvironment{}
	err := r.Get(ctx, obj, &env)

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

	logMessage(r.Log, "Reconciliation triggered", "ctrl", "app", "type", "update", "resType", "ClowdEnv", "name", a.GetName(), "namespace", a.GetNamespace())

	return reqs
}

func (r *ClowdAppReconciler) appsToEnqueueUponAppRefUpdate(ctx context.Context, a client.Object) []reconcile.Request {
	reqs := []reconcile.Request{}
	obj := types.NamespacedName{
		Name:      a.GetName(),
		Namespace: a.GetNamespace(),
	}

	// Get the ClowdAppRef resource

	appRef := crd.ClowdAppRef{}
	err := r.Get(ctx, obj, &appRef)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return reqs
		}
		r.Log.Error(err, "Failed to fetch ClowdAppRef")
		return nil
	}

	// Get all ClowdApp resources that reference this env
	appList := &crd.ClowdAppList{}
	err = r.List(ctx, appList, client.MatchingFields{"spec.envName": appRef.Spec.EnvName})
	if err != nil {
		r.Log.Error(err, "Failed to fetch ClowdApps")
		return nil
	}

	// Filter based on dependencies - only reconcile apps that have this appRef as a dependency
	for _, app := range appList.Items {
		// Check if this app depends on the appRef
		for _, dep := range append(app.Spec.Dependencies, app.Spec.OptionalDependencies...) {
			if dep == appRef.Name {
				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      app.Name,
						Namespace: app.Namespace,
					},
				})
				break
			}
		}
	}

	logMessage(r.Log, "Reconciliation triggered", "ctrl", "app", "type", "update", "resType", "ClowdAppRef", "name", a.GetName(), "namespace", a.GetNamespace())

	return reqs
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
