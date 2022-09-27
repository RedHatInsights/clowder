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

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
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

	// Import the providers to initialize them

	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
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
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
	_ "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/web"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

const (
	envFinalizer = "finalizer.env.cloud.redhat.com"
	SILENTFAIL   = "SILENTFAIL"
)

// ClowdEnvironmentReconciler reconciles a ClowdEnvironment object
type ClowdEnvironmentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	IPCCache *obj.IPCCache
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdenvironments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdenvironments/status,verbs=get;update;patch

//Acts as a guard for a reconciliation cycle, as well as initatizes a bunch of required objects
func (r *ClowdEnvironmentReconciler) getEnv(ctx context.Context, req ctrl.Request) (crd.ClowdEnvironment, ctrl.Result, error) {
	env := crd.ClowdEnvironment{}

	if getEnvErr := r.Client.Get(ctx, req.NamespacedName, &env); getEnvErr != nil {
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

//Reconcile fn
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
	cacheConfig := rc.NewCacheConfig(Scheme, errors.ClowdKey("log"), ProtectedGVKs, DebugOptions)
	cache := rc.NewObjectCache(ctx, r.Client, cacheConfig)

	r.initMetrics(env)

	defer func() {
		managedEnvsMetric.Set(float64(len(managedEnvironments)))
	}()

	configObj := r.IPCCache.GetWriteableConfig(env.Name)

	r.IPCCache.LockConfig(env.Name)
	defer r.IPCCache.UnlockConfig(env.Name)
	reconciliation := ClowdEnvironmentReconciliation{
		cache:    &cache,
		recorder: r.Recorder,
		ctx:      &ctx,
		client:   r.Client,
		env:      &env,
		log:      &log,
		config:   configObj,
	}

	result, resErr := reconciliation.Reconcile()
	if resErr != nil {
		if shouldSkipReconciliation(resErr) {
			return result, nil
		}
		return result, resErr
	}
	r.IPCCache.PersistConfig(env.Name)
	managedEnvironments[env.Name] = true

	return ctrl.Result{}, nil
}

func runProvidersForEnv(log logr.Logger, provider providers.Provider) error {
	for _, provAcc := range providers.ProvidersRegistration.Registry {
		provutils.DebugLog(log, "running provider:", "name", provAcc.Name, "order", provAcc.Order)
		start := time.Now()
		prov, err := provAcc.SetupProvider(&provider)
		prov.EnvProvide()
		elapsed := time.Since(start).Seconds()
		providerMetrics.With(prometheus.Labels{"provider": provAcc.Name, "source": "clowdenv"}).Observe(elapsed)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("getprov: %s", provAcc.Name), err)
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

// SetupWithManager sets up with manager
func (r *ClowdEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("env")

	ctrlr := ctrl.NewControllerManagedBy(mgr).For(&crd.ClowdEnvironment{})

	ctrlr.Owns(&apps.Deployment{}, builder.WithPredicates(deploymentPredicate(r.Log, "app")))
	ctrlr.Owns(&core.Service{}, builder.WithPredicates(generationOnlyPredicate(r.Log, "app")))
	ctrlr.Owns(&core.Secret{}, builder.WithPredicates(alwaysPredicate(r.Log, "app")))
	ctrlr.Watches(
		&source.Kind{Type: &crd.ClowdApp{}},
		handler.EnqueueRequestsFromMapFunc(r.envToEnqueueUponAppUpdate),
		builder.WithPredicates(generationOnlyPredicate(r.Log, "env")),
	)

	if clowderconfig.LoadedConfig.Features.WatchStrimziResources {
		ctrlr.Owns(&strimzi.Kafka{}, builder.WithPredicates(kafkaPredicate(r.Log, "app")))
		ctrlr.Owns(&strimzi.KafkaConnect{}, builder.WithPredicates(alwaysPredicate(r.Log, "app")))
		ctrlr.Owns(&strimzi.KafkaUser{}, builder.WithPredicates(alwaysPredicate(r.Log, "app")))
		ctrlr.Owns(&strimzi.KafkaTopic{}, builder.WithPredicates(alwaysPredicate(r.Log, "app")))
	}

	ctrlr.WithOptions(controller.Options{
		RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Duration(500*time.Millisecond), time.Duration(60*time.Second)),
	})
	return ctrlr.Complete(r)
}

func (r *ClowdEnvironmentReconciler) envToEnqueueUponAppUpdate(a client.Object) []reconcile.Request {
	ctx := context.Background()
	obj := types.NamespacedName{
		Name:      a.GetName(),
		Namespace: a.GetNamespace(),
	}

	// Get the ClowdEnvironment resource

	app := crd.ClowdApp{}
	err := r.Client.Get(ctx, obj, &app)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return []reconcile.Request{}
		}
		r.Log.Error(err, "Failed to fetch ClowdApp")
		return nil
	}

	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name: app.Spec.EnvName,
		},
	}}
}
