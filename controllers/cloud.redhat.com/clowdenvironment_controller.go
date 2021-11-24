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
	"sort"
	"sync"
	"time"

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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	// Import the providers to initialize them
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
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var mu sync.RWMutex
var cEnv = ""

const envFinalizer = "finalizer.env.cloud.redhat.com"

// ClowdEnvironmentReconciler reconciles a ClowdEnvironment object
type ClowdEnvironmentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdenvironments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdenvironments/status,verbs=get;update;patch

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

//Reconcile fn
func (r *ClowdEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("env", req.Name).WithValues("id", utils.RandString(5))
	ctx = context.WithValue(ctx, errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)
	env := crd.ClowdEnvironment{}
	err := r.Client.Get(ctx, req.NamespacedName, &env)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	GetEnvResourceStatus(ctx, r.Client, &env)
	isEnvMarkedForDeletion := env.GetDeletionTimestamp() != nil
	if isEnvMarkedForDeletion {
		if contains(env.GetFinalizers(), envFinalizer) {
			if err := r.finalizeEnvironment(log, &env); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&env, envFinalizer)
			err := r.Update(ctx, &env)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(env.GetFinalizers(), envFinalizer) {
		if err := r.addFinalizer(log, &env); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("Reconciliation started", "env", env.Name)
	requestMetrics.With(prometheus.Labels{"type": "env", "name": env.Name}).Inc()

	if env.Spec.Disabled {
		log.Info("Reconciliation aborted - set to be disabled", "env", env.Name)
		return ctrl.Result{}, nil
	}

	if env.Status.TargetNamespace == "" {
		if env.Spec.TargetNamespace != "" {
			namespace := core.Namespace{}
			namespaceName := types.NamespacedName{
				Name: env.Spec.TargetNamespace,
			}
			err := r.Client.Get(ctx, namespaceName, &namespace)
			if err != nil {
				r.Recorder.Eventf(&env, "Warning", "NamespaceMissing", "Requested Target Namespace [%s] is missing", env.Spec.TargetNamespace)
				SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, err)
				return ctrl.Result{Requeue: true}, err
			}
			env.Status.TargetNamespace = env.Spec.TargetNamespace
		} else {
			env.Status.TargetNamespace = env.GenerateTargetNamespace()
			namespace := &core.Namespace{}
			namespace.SetName(env.Status.TargetNamespace)
			err := r.Client.Create(ctx, namespace)
			if err != nil {
				SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, err)
				return ctrl.Result{Requeue: true}, err
			}
		}
		err := r.Client.Status().Update(ctx, &env)
		if err != nil {
			SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, err)
			return ctrl.Result{Requeue: true}, err
		}
	}

	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &env)

	cache := providers.NewObjectCache(ctx, r.Client, scheme)

	provider := providers.Provider{
		Ctx:    ctx,
		Client: r.Client,
		Env:    &env,
		Cache:  &cache,
		Log:    log,
	}

	SetEnv(env.Name)
	defer ReleaseEnv()
	provErr := runProvidersForEnv(log, provider)

	if provErr != nil {
		err := SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, provErr)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, provErr
	}

	cacheErr := cache.ApplyAll()

	if cacheErr != nil {
		err := SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, cacheErr)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, cacheErr
	}

	if err == nil {
		if _, ok := managedEnvironments[env.Name]; !ok {
			managedEnvironments[env.Name] = true
		}
		managedEnvsMetric.Set(float64(len(managedEnvironments)))
	}

	err = r.setAppInfo(provider)
	if err != nil {
		SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, err)
		return ctrl.Result{Requeue: true}, err
	}

	if statusErr := SetEnvResourceStatus(ctx, r.Client, &env); statusErr != nil {
		SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, err)
		return ctrl.Result{Requeue: true}, err
	}

	setPrometheusStatus(&env)

	ready, err := GetEnvResourceStatus(ctx, r.Client, &env)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	env.Status.Ready = ready
	env.Status.Generation = env.Generation

	if err := r.Client.Status().Update(ctx, &env); err != nil {
		SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationFailed, err)
		return ctrl.Result{Requeue: true}, err
	}

	// Delete all resources that are not used anymore
	rErr := cache.Reconcile(&env)
	if rErr != nil {
		return ctrl.Result{Requeue: true}, rErr
	}

	err = SetClowdEnvConditions(ctx, r.Client, &env, crd.ReconciliationSuccessful, nil)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	r.Recorder.Eventf(&env, "Normal", "SuccessfulReconciliation", "Environment reconciled [%s]", env.GetClowdName())
	log.Info("Reconciliation successful", "env", env.Name)

	return ctrl.Result{}, nil
}

func setPrometheusStatus(env *crd.ClowdEnvironment) {
	var hostname string

	if env.Spec.Providers.Metrics.Mode == "app-interface" {
		hostname = env.Spec.Providers.Metrics.Prometheus.AppInterfaceHostname
	} else {
		hostname = fmt.Sprintf("prometheus-operated.%s.svc.cluster.local", env.Status.TargetNamespace)
	}

	env.Status.Prometheus = crd.PrometheusStatus{Hostname: hostname}
}

func runProvidersForEnv(log logr.Logger, provider providers.Provider) error {
	for _, provAcc := range providers.ProvidersRegistration.Registry {
		log.Info("running provider:", "name", provAcc.Name, "order", provAcc.Order)
		start := time.Now()
		_, err := provAcc.SetupProvider(&provider)
		elapsed := time.Since(start).Seconds()
		providerMetrics.With(prometheus.Labels{"provider": provAcc.Name, "source": "clowdenv"}).Observe(elapsed)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("getprov: %s", provAcc.Name), err)
		}
		log.Info("running provider: complete", "name", provAcc.Name, "order", provAcc.Order, "elapsed", fmt.Sprintf("%f", elapsed))
	}
	return nil
}

// SetupWithManager sets up with manager
func (r *ClowdEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("env")

	ctrlr := ctrl.NewControllerManagedBy(mgr).
		Owns(&apps.Deployment{}, builder.WithPredicates(ignoreStatusUpdatePredicate(r.Log, "app"))).
		Owns(&core.Service{}, builder.WithPredicates(ignoreStatusUpdatePredicate(r.Log, "app"))).
		Watches(
			&source.Kind{Type: &crd.ClowdApp{}},
			handler.EnqueueRequestsFromMapFunc(r.envToEnqueueUponAppUpdate),
			builder.WithPredicates(ignoreStatusUpdatePredicate(r.Log, "env")),
		).
		For(&crd.ClowdEnvironment{})

	if clowderconfig.LoadedConfig.Features.WatchStrimziResources {
		ctrlr.Owns(&strimzi.Kafka{})
		ctrlr.Owns(&strimzi.KafkaConnect{})
		ctrlr.Owns(&strimzi.KafkaUser{})
		ctrlr.Owns(&strimzi.KafkaTopic{})
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

func (r *ClowdEnvironmentReconciler) setAppInfo(p providers.Provider) error {
	// Get all the ClowdApp resources
	appList, err := p.Env.GetAppsInEnv(p.Ctx, p.Client)

	if err != nil {
		return err
	}
	apps := []crd.AppInfo{}

	appMap := map[string]crd.ClowdApp{}
	names := []string{}

	for _, app := range appList.Items {
		name := fmt.Sprintf("%s-%s", app.Name, app.Namespace)
		names = append(names, name)
		appMap[name] = app
	}

	sort.Strings(names)

	// Populate
	for _, name := range names {
		app := appMap[name]

		if app.GetDeletionTimestamp() != nil {
			continue
		}

		appstatus := crd.AppInfo{
			Name:        app.Name,
			Deployments: []crd.DeploymentInfo{},
		}

		depMap := map[string]crd.Deployment{}
		depNames := []string{}

		for _, pod := range app.Spec.Deployments {
			depNames = append(depNames, pod.Name)
			depMap[pod.Name] = pod
		}

		sort.Strings(depNames)

		for _, podName := range depNames {
			pod := depMap[podName]

			deploymentStatus := crd.DeploymentInfo{
				Name: fmt.Sprintf("%s-%s", app.Name, pod.Name),
			}
			if bool(pod.Web) || pod.WebServices.Public.Enabled {
				deploymentStatus.Hostname = fmt.Sprintf("%s.%s.svc", deploymentStatus.Name, app.Namespace)
				deploymentStatus.Port = p.Env.Spec.Providers.Web.Port
			}
			appstatus.Deployments = append(appstatus.Deployments, deploymentStatus)
		}
		apps = append(apps, appstatus)
	}

	p.Env.Status.Apps = apps
	return nil
}

func (r *ClowdEnvironmentReconciler) finalizeEnvironment(reqLogger logr.Logger, e *crd.ClowdEnvironment) error {

	delete(managedEnvironments, e.Name)

	if e.Spec.TargetNamespace == "" {
		namespace := &core.Namespace{}
		namespace.SetName(e.Status.TargetNamespace)
		reqLogger.Info(fmt.Sprintf("Removing auto-generated namespace for %s", e.Name))
		r.Recorder.Eventf(e, "Warning", "NamespaceDeletion", "Clowder Environment [%s] had no targetNamespace, deleting generated namespace", e.Name)
		r.Delete(context.TODO(), namespace)
	}
	managedEnvsMetric.Set(float64(len(managedEnvironments)))
	reqLogger.Info("Successfully finalized ClowdEnvironment")
	return nil
}

func (r *ClowdEnvironmentReconciler) addFinalizer(reqLogger logr.Logger, e *crd.ClowdEnvironment) error {
	reqLogger.Info("Adding Finalizer for the ClowdEnvironment")
	controllerutil.AddFinalizer(e, envFinalizer)

	// Update CR
	err := r.Update(context.TODO(), e)
	if err != nil {
		reqLogger.Error(err, "Failed to update ClowdEnvironment with finalizer")
		return err
	}
	return nil
}
