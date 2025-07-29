package controllers

import (
	"context"
	"fmt"
	"sort"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	cond "sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// SkippedError represents an error that occurred during reconciliation that can be skipped
type SkippedError struct {
	err error
}

func (se SkippedError) Error() string {
	return fmt.Sprintf("skipped because: %s", se.err.Error())
}

// NewSkippedError creates a new SkippedError with the given message
func NewSkippedError(errString string) error {
	return SkippedError{err: fmt.Errorf("%s", errString)}
}

const (
	SKIPRECONCILE = "SKIPRECONCILE"
)

// During reconciliation we handle errors in 2 ways: sometimes we want to error out of reconciliation and sometimes we want to skip reconciliation.
func shouldSkipReconciliation(err error) bool {
	_, isSkippedError := err.(SkippedError)
	if err != nil && isSkippedError {
		return true
	}
	return false
}

// ClowdEnvironmentReconciliation represents a single reconciliation event
// This type is created by ClowdEnvironmentReconciler which handles the reconciliation cycle as a whole
// ClowdEnvironmentReconciliation encapsulates all of the state and logic requires for a single
// reconciliation event
type ClowdEnvironmentReconciliation struct {
	cache     *rc.ObjectCache
	recorder  record.EventRecorder
	ctx       context.Context
	client    client.Client
	env       *crd.ClowdEnvironment
	log       *logr.Logger
	oldStatus *crd.ClowdEnvironmentStatus
	hashCache *hashcache.HashCache
}

// Returns a list of step methods that should be run during reconciliation
func (r *ClowdEnvironmentReconciliation) steps() []func() (ctrl.Result, error) {
	return []func() (ctrl.Result, error){
		r.markedForDeletion,
		r.addFinalizerIfRequired,
		r.perProviderMetrics,
		r.setToBeDisabled,
		r.initTargetNamespace,
		r.isTargetNamespaceMarkedForDeletion,
		r.runProviders,
		r.applyCache,
		r.setAppInfo,
		r.setEnvResourceStatus,
		r.setPrometheusStatus,
		r.setEnvStatus,
		r.finalStatusError,
		r.deleteUnusedResources,
		r.setClowdEnvConditions,
		r.logSuccess,
	}
}

// Reconcile iterates through the steps of the reconciliation process
func (r *ClowdEnvironmentReconciliation) Reconcile() (ctrl.Result, error) {
	r.log.Info("Reconciliation started")
	// The env stays locked for the entire reconciliation
	// This is a slight change from the previous implementation
	// where the lock wasn't initated until the target namespace had been initialized
	SetEnv(r.env.Name)
	defer ReleaseEnv()
	for _, step := range r.steps() {
		result, err := step()
		if err != nil {
			return result, err
		}
	}

	return ctrl.Result{}, nil
}

// Determine if app is marked for deletion, and if so finalize and end resonciliation
func (r *ClowdEnvironmentReconciliation) markedForDeletion() (ctrl.Result, error) {
	isEnvMarkedForDeletion := r.env.GetDeletionTimestamp() != nil
	if isEnvMarkedForDeletion {
		if contains(r.env.GetFinalizers(), envFinalizer) {
			if finalizeErr := r.finalizeEnvironmentImplementation(); finalizeErr != nil {
				r.log.Info("Cloud not finalize", "err", finalizeErr)
				return ctrl.Result{Requeue: true}, finalizeErr
			}

			controllerutil.RemoveFinalizer(r.env, envFinalizer)
			removeFinalizeErr := r.client.Update(r.ctx, r.env)
			if removeFinalizeErr != nil {
				r.log.Info("Cloud not remove finalizer", "err", removeFinalizeErr)
				return ctrl.Result{}, removeFinalizeErr
			}
			return ctrl.Result{}, NewSkippedError("env is marked for delete and finalizers removed")
		}
		return ctrl.Result{}, NewSkippedError("env is marked for delete and has no finalizer")
	}
	return ctrl.Result{}, nil
}

// Perform finalization of the environment. Called by markedForDeleteion
// Note: _ at beginning of method name indicates that a method is called
// by a step method, not directly by the reconciliation loop
func (r *ClowdEnvironmentReconciliation) finalizeEnvironmentImplementation() error {

	provider := providers.Provider{
		Ctx:       r.ctx,
		Client:    r.client,
		Env:       r.env,
		Cache:     r.cache,
		Log:       *r.log,
		HashCache: r.hashCache,
	}

	err := runProvidersForEnvFinalize(*r.log, provider)
	if err != nil {
		return err
	}

	if r.env.Spec.TargetNamespace == "" {
		namespace := &core.Namespace{}
		namespace.SetName(r.env.Status.TargetNamespace)
		r.log.Info(fmt.Sprintf("Removing auto-generated namespace for %s", r.env.Name))
		r.recorder.Eventf(r.env, "Warning", "NamespaceDeletion", "Clowder Environment [%s] had no targetNamespace, deleting generated namespace", r.env.Name)
		_ = r.client.Delete(context.TODO(), namespace)
	}
	delete(managedEnvironments, r.env.Name)
	managedEnvsMetric.Set(float64(len(managedEnvironments)))

	delete(presentEnvironments, r.env.Name)
	presentEnvsMetric.Set(float64(len(presentEnvironments)))

	r.log.Info("Successfully finalized ClowdEnvironment")
	return nil
}

// Adds a finalizer to the environment if one is not already present
func (r *ClowdEnvironmentReconciliation) addFinalizerIfRequired() (ctrl.Result, error) {
	// Add finalizer for this CR
	if !contains(r.env.GetFinalizers(), envFinalizer) {
		if addFinalizeErr := r.addFinalizerImplementation(); addFinalizeErr != nil {
			r.log.Info("Cloud not add finalizer", "err", addFinalizeErr)
			return ctrl.Result{}, addFinalizeErr
		}
	}
	return ctrl.Result{}, nil
}

// Implementation method for addining a finalizer
func (r *ClowdEnvironmentReconciliation) addFinalizerImplementation() error {
	r.log.Info("Adding Finalizer for the ClowdEnvironment")
	controllerutil.AddFinalizer(r.env, envFinalizer)

	// Update CR
	err := r.client.Update(context.TODO(), r.env)
	if err != nil {
		r.log.Error(err, "Failed to update ClowdEnvironment with finalizer")
		return err
	}
	return nil
}

// Request per provider methods if the config calls for it
func (r *ClowdEnvironmentReconciliation) perProviderMetrics() (ctrl.Result, error) {
	if clowderconfig.LoadedConfig.Features.PerProviderMetrics {
		requestMetrics.With(prometheus.Labels{"type": "env", "name": r.env.Name}).Inc()
	}
	return ctrl.Result{}, nil
}

// Get or create and then update the cache for the target namespace
func (r *ClowdEnvironmentReconciliation) initTargetNamespace() (ctrl.Result, error) {
	if r.env.Status.TargetNamespace != "" {
		return ctrl.Result{}, nil
	}

	var result ctrl.Result
	var err error

	if r.env.Spec.TargetNamespace == "" {
		result, err = r.makeTargetNamespace()
	} else {
		result, err = r.getTargetNamespace()
	}
	if err != nil {
		return result, err
	}

	result, err = r.updateTargetNamespace()

	return result, err
}

// Get the env target namespace
func (r *ClowdEnvironmentReconciliation) getTargetNamespace() (ctrl.Result, error) {
	namespace := core.Namespace{}
	namespaceName := types.NamespacedName{
		Name: r.env.Spec.TargetNamespace,
	}
	if nErr := r.client.Get(r.ctx, namespaceName, &namespace); nErr != nil {
		r.log.Info("Namespace get error", "err", nErr)
		r.recorder.Eventf(r.env, "Warning", "NamespaceMissing", "Requested Target Namespace [%s] is missing", r.env.Spec.TargetNamespace)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, nErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, nErr
	}
	r.env.Status.TargetNamespace = r.env.Spec.TargetNamespace
	return ctrl.Result{}, nil
}

// Make a new target namespace
func (r *ClowdEnvironmentReconciliation) makeTargetNamespace() (ctrl.Result, error) {
	r.env.Status.TargetNamespace = r.env.GenerateTargetNamespace()
	namespace := &core.Namespace{}
	namespace.SetName(r.env.Status.TargetNamespace)
	if snErr := r.client.Create(r.ctx, namespace); snErr != nil {
		r.log.Info("Namespace create error", "err", snErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, snErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, snErr
	}
	return ctrl.Result{}, nil
}

// Update the target namespace
func (r *ClowdEnvironmentReconciliation) updateTargetNamespace() (ctrl.Result, error) {
	if statErr := r.client.Status().Update(r.ctx, r.env); statErr != nil {
		r.log.Info("Namespace create error", "err", statErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, statErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, statErr
	}
	return ctrl.Result{}, nil
}

// Determine if the target namespace is marked for deletion
// If it is then we enter into the SKIPRECONCILE corner case
// which is a special case where the reconciliation controller bails early
// from reconcile WIHTOUT returning an error
func (r *ClowdEnvironmentReconciliation) isTargetNamespaceMarkedForDeletion() (ctrl.Result, error) {
	ens := &core.Namespace{}
	if getNSErr := r.client.Get(r.ctx, types.NamespacedName{Name: r.env.Status.TargetNamespace}, ens); getNSErr != nil {
		r.log.Info("Get namespace error", "err", getNSErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, getNSErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, getNSErr
	}

	if ens.DeletionTimestamp != nil {
		return ctrl.Result{}, NewSkippedError("target namespace is to be deleted")
	}

	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) runProviders() (ctrl.Result, error) {
	r.hashCache.RemoveClowdObjectFromObjects(r.env)

	provider := providers.Provider{
		Ctx:       r.ctx,
		Client:    r.client,
		Env:       r.env,
		Cache:     r.cache,
		Log:       *r.log,
		HashCache: r.hashCache,
	}
	provErr := runProvidersForEnv(*r.log, provider)

	if provErr != nil {
		r.log.Info("Prov err", "err", provErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, provErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, provErr
	}

	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) applyCache() (ctrl.Result, error) {
	cacheErr := r.cache.ApplyAll()
	if cacheErr != nil {
		r.log.Info("Cache error", "err", cacheErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, cacheErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, cacheErr
	}
	return ctrl.Result{}, nil
}

// Sets info for the apps in the environment
// This method is the step and contains most of the error handling, logging etc
// The full implementation is pushed out into another method
func (r *ClowdEnvironmentReconciliation) setAppInfo() (ctrl.Result, error) {
	if setAppErr := r.setAppInfoImplementation(); setAppErr != nil {
		r.log.Info("setAppInfo error", "err", setAppErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, setAppErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, setAppErr
	}
	return ctrl.Result{}, nil
}

// Performs the actual business logic for the setAppInfo step
// TODO this is a bear of a method. Factor this out into pieces.
func (r *ClowdEnvironmentReconciliation) setAppInfoImplementation() error {

	// Get all the ClowdApp resources
	appList, err := r.env.GetAppsInEnv(r.ctx, r.client)

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
				deploymentStatus.Port = r.env.Spec.Providers.Web.Port
			}
			appstatus.Deployments = append(appstatus.Deployments, deploymentStatus)
		}
		apps = append(apps, appstatus)
	}

	r.env.Status.Apps = apps
	return nil
}

func (r *ClowdEnvironmentReconciliation) setEnvResourceStatus() (ctrl.Result, error) {
	if statusErr := SetEnvResourceStatus(r.ctx, r.client, r.env); statusErr != nil {
		r.log.Info("SetEnvResourceStatus error", "err", statusErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, statusErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, statusErr
	}
	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) setPrometheusStatus() (ctrl.Result, error) {
	var url string

	if r.env.Spec.Providers.Metrics.Mode == "app-interface" {
		url = r.env.Spec.Providers.Metrics.Prometheus.AppInterfaceInternalURL
	} else {
		url = fmt.Sprintf("http://prometheus-operated.%s.svc.cluster.local:9090", r.env.Status.TargetNamespace)
	}

	r.env.Status.Prometheus = crd.PrometheusStatus{ServerAddress: url}

	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) setEnvStatus() (ctrl.Result, error) {
	envReady, _, getEnvResErr := GetEnvResourceStatus(r.ctx, r.client, r.env)
	if getEnvResErr != nil {
		r.log.Info("GetEnvResourceStatus error", "err", getEnvResErr)
		return ctrl.Result{Requeue: true}, getEnvResErr
	}

	envStatus := core.ConditionFalse
	successCond := cond.Get(r.env, crd.ReconciliationSuccessful)
	if successCond != nil {
		envStatus = successCond.Status
	}

	r.env.Status.Ready = envReady && (envStatus == core.ConditionTrue)
	r.env.Status.Generation = r.env.Generation

	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) finalStatusError() (ctrl.Result, error) {
	if finalStatusErr := r.client.Status().Update(r.ctx, r.env); finalStatusErr != nil {
		r.log.Info("Final Status error", "err", finalStatusErr)
		if setClowdStatusErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationFailed, r.oldStatus, finalStatusErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, finalStatusErr
	}
	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) deleteUnusedResources() (ctrl.Result, error) {
	opts := []client.ListOption{
		client.MatchingLabels{r.env.GetPrimaryLabel(): r.env.GetClowdName()},
	}

	// Delete all resources that are not used anymore
	rErr := r.cache.Reconcile(r.env.GetUID(), opts...)
	if rErr != nil {
		return ctrl.Result{Requeue: true}, rErr
	}

	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) setClowdEnvConditions() (ctrl.Result, error) {
	if successSetErr := SetClowdEnvConditions(r.ctx, r.client, r.env, crd.ReconciliationSuccessful, r.oldStatus, nil); successSetErr != nil {
		r.log.Info("Set status error", "err", successSetErr)
		return ctrl.Result{Requeue: true}, successSetErr
	}
	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) logSuccess() (ctrl.Result, error) {
	r.recorder.Eventf(r.env, "Normal", "SuccessfulReconciliation", "Environment reconciled [%s]", r.env.GetClowdName())
	r.log.Info("Reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *ClowdEnvironmentReconciliation) setToBeDisabled() (ctrl.Result, error) {
	if r.env.Spec.Disabled {
		return ctrl.Result{}, NewSkippedError("env is disabled")
	}
	return ctrl.Result{}, nil
}
