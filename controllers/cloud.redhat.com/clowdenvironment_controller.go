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
	"sync"
	"time"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	// Import the providers to initialize them
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"

	// These blank imports make the providers go wheeeeee
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

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

var mu sync.RWMutex
var cEnv = ""

const (
	envFinalizer = "finalizer.env.cloud.redhat.com"
	SILENTFAIL   = "SILENTFAIL"
)

// ClowdEnvironmentReconciler reconciles a ClowdEnvironment object
type ClowdEnvironmentReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	HashCache *hashcache.HashCache
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdenvironments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdenvironments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapprefs,verbs=get;list;watch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapprefs/status,verbs=get;update;patch

func SetEnv(name string) {
	mu.Lock()
	defer mu.Unlock()
	cEnv = name
}

func ReleaseEnv() {
	mu.Lock()
	defer mu.Unlock()
	cEnv = ""
}

func ReadEnv() string {
	mu.RLock()
	defer mu.RUnlock()
	return cEnv
}

// Acts as a guard for a reconciliation cycle, as well as initatizes a bunch of required objects
func (r *ClowdEnvironmentReconciler) getEnv(ctx context.Context, req ctrl.Request) (crd.ClowdEnvironment, ctrl.Result, error) {
	env := crd.ClowdEnvironment{}

	if getEnvErr := r.Get(ctx, req.NamespacedName, &env); getEnvErr != nil {
		return env, ctrl.Result{}, getEnvErr
	}

	return env, ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciler) initMetrics(env crd.ClowdEnvironment) {
	if _, ok := presentEnvironments[env.Name]; !ok {
		presentEnvironments[env.Name] = true
	}
	presentEnvsMetric.Set(float64(len(presentEnvironments)))

	delete(managedEnvironments, env.Name)
}

// Reconcile fn
func (r *ClowdEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("env", req.Name).WithValues("rid", utils.RandString(5))
	ctx = context.WithValue(ctx, errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)

	env, res, err := r.getEnv(ctx, req)
	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		log.Info("Namespace not found", "err", err)
		return res, err
	}

	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &env)
	cacheConfig := rc.NewCacheConfig(Scheme, nil, ProtectedGVKs, rc.Options{StrictGVK: true, DebugOptions: DebugOptions, Ordering: applyOrder})
	cache := rc.NewObjectCache(ctx, r.Client, &log, cacheConfig)

	r.initMetrics(env)

	defer func() {
		managedEnvsMetric.Set(float64(len(managedEnvironments)))
	}()

	reconciliation := ClowdEnvironmentReconciliation{
		cache:     &cache,
		recorder:  r.Recorder,
		ctx:       ctx,
		client:    r.Client,
		env:       &env,
		log:       &log,
		oldStatus: env.Status.DeepCopy(),
		hashCache: r.HashCache,
	}

	result, resErr := reconciliation.Reconcile()
	if resErr != nil {
		if shouldSkipReconciliation(resErr) {
			log.Info("skipping", "error", resErr.Error(), "skipping", "true", "requeue", result.Requeue)
			return result, nil
		}
		log.Error(err, "error in reconciliation", "skipping", "false", "requeue", result.Requeue)
		return result, resErr
	}
	managedEnvironments[env.Name] = true

	return ctrl.Result{}, nil
}

func runProvidersForEnv(log logr.Logger, provider providers.Provider) error {
	for _, provAcc := range providers.ProvidersRegistration.Registry {
		provutils.DebugLog(log, "running provider:", "name", provAcc.Name, "order", provAcc.Order)
		start := time.Now()
		prov, err := provAcc.SetupProvider(&provider)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("getprov: %s", provAcc.Name), err)
		}
		err = prov.EnvProvide()
		elapsed := time.Since(start).Seconds()
		providerMetrics.With(prometheus.Labels{"provider": provAcc.Name, "source": "clowdenv"}).Observe(elapsed)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("runprov: %s", provAcc.Name), err)
		}
		provutils.DebugLog(log, "running provider: complete", "name", provAcc.Name, "order", provAcc.Order, "elapsed", fmt.Sprintf("%f", elapsed))
	}
	return nil
}

func runProvidersForEnvFinalize(log logr.Logger, provider providers.Provider) error {
	for _, provAcc := range providers.ProvidersRegistration.Registry {
		if provAcc.FinalizeProvider != nil {
			provutils.DebugLog(log, "running provider finalize:", "name", provAcc.Name, "order", provAcc.Order)
			err := provAcc.FinalizeProvider(&provider)
			if err != nil {
				return errors.Wrap(fmt.Sprintf("prov finalize: %s", provAcc.Name), err)
			}
			provutils.DebugLog(log, "running provider finalize: complete", "name", provAcc.Name, "order", provAcc.Order)
		}
	}
	return nil
}

func (r *ClowdEnvironmentReconciler) setupWatch(ctrlr *builder.Builder, mgr ctrl.Manager, obj client.Object, handlerBuilder HandlerFuncBuilder) error {
	handler, err := createNewHandler(mgr, r.Scheme, handlerBuilder, r.Log, "app", &crd.ClowdEnvironment{}, r.HashCache)
	if err != nil {
		return err
	}
	ctrlr.Watches(obj, handler)

	return nil
}

// SetupWithManager sets up with manager
func (r *ClowdEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("env")

	cache := mgr.GetCache()

	// Index ClowdAppRef by envName for efficient lookups
	if err := cache.IndexField(
		context.TODO(), &crd.ClowdAppRef{}, "spec.envName", func(o client.Object) []string {
			return []string{o.(*crd.ClowdAppRef).Spec.EnvName}
		}); err != nil {
		return err
	}

	ctrlr := ctrl.NewControllerManagedBy(mgr).For(&crd.ClowdEnvironment{})

	watchers := []Watcher{
		{obj: &apps.Deployment{}, filter: deploymentFilter},
		{obj: &core.Service{}, filter: alwaysFilter},
		{obj: &core.Secret{}, filter: alwaysFilter},
	}

	if clowderconfig.LoadedConfig.Features.WatchStrimziResources {
		watchers = append(watchers, Watcher{obj: &strimzi.Kafka{}, filter: kafkaFilter})
		watchers = append(watchers, Watcher{obj: &strimzi.KafkaConnect{}, filter: alwaysFilter})
		watchers = append(watchers, Watcher{obj: &strimzi.KafkaUser{}, filter: alwaysFilter})
		watchers = append(watchers, Watcher{obj: &strimzi.KafkaTopic{}, filter: alwaysFilter})
	}

	for _, watcher := range watchers {
		err := r.setupWatch(ctrlr, mgr, watcher.obj, watcher.filter)
		if err != nil {
			return err
		}
	}

	ctrlr.Watches(
		&crd.ClowdApp{},
		handler.EnqueueRequestsFromMapFunc(r.envToEnqueueUponAppUpdate),
		builder.WithPredicates(predicate.GenerationChangedPredicate{}),
	)

	ctrlr.Watches(
		&crd.ClowdAppRef{},
		handler.EnqueueRequestsFromMapFunc(r.envToEnqueueUponAppRefUpdate),
		builder.WithPredicates(predicate.GenerationChangedPredicate{}),
	)

	ctrlr.WithOptions(controller.Options{
		RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Duration(500*time.Millisecond), time.Duration(60*time.Second)),
	})
	return ctrlr.Complete(r)
}

func (r *ClowdEnvironmentReconciler) envToEnqueueUponAppUpdate(ctx context.Context, a client.Object) []reconcile.Request {
	obj := types.NamespacedName{
		Name:      a.GetName(),
		Namespace: a.GetNamespace(),
	}

	// Get the ClowdEnvironment resource

	app := crd.ClowdApp{}
	err := r.Get(ctx, obj, &app)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return []reconcile.Request{}
		}
		r.Log.Error(err, "Failed to fetch ClowdApp")
		return nil
	}

	logMessage(r.Log, "Reconciliation triggered", "ctrl", "env", "type", "update", "resType", "ClowdApp", "name", a.GetName(), "namespace", a.GetNamespace())

	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name: app.Spec.EnvName,
		},
	}}
}

func (r *ClowdEnvironmentReconciler) envToEnqueueUponAppRefUpdate(ctx context.Context, a client.Object) []reconcile.Request {
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
			return []reconcile.Request{}
		}
		r.Log.Error(err, "Failed to fetch ClowdAppRef")
		return nil
	}

	logMessage(r.Log, "Reconciliation triggered", "ctrl", "env", "type", "update", "resType", "ClowdAppRef", "name", a.GetName(), "namespace", a.GetNamespace())

	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name: appRef.Spec.EnvName,
		},
	}}
}
