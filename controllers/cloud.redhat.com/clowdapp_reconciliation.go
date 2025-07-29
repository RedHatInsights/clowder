package controllers

import (
	"context"
	"fmt"
	"time"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
)

type ClowdAppReconciliation struct {
	cache                 *rc.ObjectCache
	recorder              record.EventRecorder
	ctx                   context.Context
	client                client.Client
	log                   *logr.Logger
	app                   *crd.ClowdApp
	req                   *ctrl.Request
	env                   *crd.ClowdEnvironment
	reconciliationMetrics ReconciliationMetrics
	config                *config.AppConfig
	oldStatus             *crd.ClowdAppStatus
	hashCache             *hashcache.HashCache
}

func (r *ClowdAppReconciliation) steps() []func() (ctrl.Result, error) {
	return []func() (ctrl.Result, error){
		r.getApp,
		r.setPresentAndManagedApps,
		r.startMetrics,
		r.isAppMarkedForDeletion,
		r.addFinalizer,
		r.isEnvLocked,
		r.isAppDisabled,
		r.isAppNamespaceDeleted,
		r.getClowdEnv,
		r.isEnvNamespaceDeleted,
		r.isClowdEnvReconciled,
		r.isEnvReady,
		r.createCache,
		r.runProviders,
		r.applyCache,
		r.setAppResourceStatus,
		r.deletedUnusedResources,
		r.setReconciliationSuccessful,
		r.stopMetrics,
	}
}

func (r *ClowdAppReconciliation) Reconcile() (ctrl.Result, error) {
	for _, step := range r.steps() {
		result, err := step()
		if err != nil {
			return result, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) startMetrics() (ctrl.Result, error) {
	r.reconciliationMetrics = ReconciliationMetrics{}
	r.reconciliationMetrics.init(r.app.Name, r.app.Spec.EnvName)
	r.reconciliationMetrics.start()

	if clowderconfig.LoadedConfig.Features.PerProviderMetrics {
		requestMetrics.With(prometheus.Labels{"type": "app", "name": r.app.GetIdent()}).Inc()
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) stopMetrics() (ctrl.Result, error) {
	r.reconciliationMetrics.stop()
	return ctrl.Result{}, nil
}

func ReportDependencies(ctx context.Context, pClient client.Client, app *crd.ClowdApp, env *crd.ClowdEnvironment) error {
	appName := app.Name
	appDependencies := app.Spec.Dependencies
	appDependencies = append(appDependencies, app.Spec.OptionalDependencies...)

	// Don't record metrics if not enabled
	if !clowderconfig.LoadedConfig.Features.EnableDependencyMetrics {
		return nil
	}

	applist, err := env.GetAppsInEnv(ctx, pClient)
	if err != nil {
		return err
	}

	for _, dependency := range appDependencies {
		for _, app := range applist.Items {
			if app.Name != dependency {
				continue
			}

			observedReady := 0.0
			if app.Status.Ready {
				observedReady = 1.0
			}

			dependencyMetrics.With(prometheus.Labels{"app": appName, "dependency": dependency}).Set(observedReady)

		}
	}

	return nil
}

func (r *ClowdAppReconciliation) setPresentAndManagedApps() (ctrl.Result, error) {
	presentApps[r.app.GetIdent()] = true

	presentAppsMetric.Set(float64(len(presentApps)))

	delete(managedApps, r.app.GetIdent())

	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) getApp() (ctrl.Result, error) {
	if getAppErr := r.client.Get(r.ctx, r.req.NamespacedName, r.app); getAppErr != nil {
		if k8serr.IsNotFound(getAppErr) {
			// Must have been deleted
			return ctrl.Result{}, NewSkippedError("app is deleted")
		}
		r.log.Info("App not found", "env", r.app.Spec.EnvName, "app", r.app.GetIdent(), "err", getAppErr)
		return ctrl.Result{}, getAppErr
	}
	// This is kinda side-effecty but I couldn't think of a better place
	// to put it.
	logWithEnv := r.log.WithValues("env", r.app.Spec.EnvName)
	r.log = &logWithEnv

	r.oldStatus = r.app.Status.DeepCopy()

	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) isAppMarkedForDeletion() (ctrl.Result, error) {
	isAppMarkedForDeletion := r.app.GetDeletionTimestamp() != nil
	if isAppMarkedForDeletion {
		if contains(r.app.GetFinalizers(), appFinalizer) {
			if finalizeErr := r.finalizeApp(); finalizeErr != nil {
				r.log.Info("Cloud not finalize", "err", finalizeErr)
				return ctrl.Result{}, finalizeErr
			}

			controllerutil.RemoveFinalizer(r.app, appFinalizer)
			removeFinalizeErr := r.client.Update(r.ctx, r.app)
			if removeFinalizeErr != nil {
				r.log.Info("Cloud not remove finalizer", "err", removeFinalizeErr)
				return ctrl.Result{}, removeFinalizeErr
			}
			return ctrl.Result{}, NewSkippedError("app is marked for delete and finalizers removed")
		}
		return ctrl.Result{}, NewSkippedError("app is marked for delete and has no finalizer")
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) finalizeApp() error {
	// We remove it from the managed list because it may have been managed before, but it may not be after this reconcile.
	delete(managedApps, r.app.GetIdent())
	managedAppsMetric.Set(float64(len(managedApps)))

	delete(presentApps, r.app.GetIdent())
	presentAppsMetric.Set(float64(len(presentApps)))

	r.log.Info("Successfully finalized ClowdApp")
	return nil
}

func (r *ClowdAppReconciliation) addFinalizer() (ctrl.Result, error) {
	if !contains(r.app.GetFinalizers(), appFinalizer) {
		if addFinalizeErr := r.addFinalizerImplementation(); addFinalizeErr != nil {
			r.log.Info("Cloud not add finalizer", "err", addFinalizeErr)
			return ctrl.Result{}, addFinalizeErr
		}
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) addFinalizerImplementation() error {
	r.log.Info("Adding Finalizer for the ClowdApp")
	controllerutil.AddFinalizer(r.app, appFinalizer)

	// Update CR
	err := r.client.Update(r.ctx, r.app)
	if err != nil {
		r.log.Error(err, "Failed to update ClowdApp with finalizer")
		return err
	}
	return nil
}

func (r *ClowdAppReconciliation) isEnvLocked() (ctrl.Result, error) {
	if ReadEnv() == r.app.Spec.EnvName {
		r.recorder.Eventf(r.app, "Warning", "ClowdEnvLocked", "Clowder Environment [%s] is locked", r.app.Spec.EnvName)
		return ctrl.Result{Requeue: true}, NewSkippedError("env is locked")
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) isAppDisabled() (ctrl.Result, error) {
	if r.app.Spec.Disabled {
		return ctrl.Result{}, NewSkippedError("app disabled")
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) getClowdEnv() (ctrl.Result, error) {
	updatedContext := context.WithValue(r.ctx, errors.ClowdKey("obj"), r.app)
	r.ctx = updatedContext
	r.env = &crd.ClowdEnvironment{}

	if getEnvErr := r.client.Get(r.ctx, types.NamespacedName{Name: r.app.Spec.EnvName}, r.env); getEnvErr != nil {
		r.log.Info("ClowdEnv missing", "err", getEnvErr)
		r.recorder.Eventf(r.app, "Warning", "ClowdEnvMissing", "Clowder Environment [%s] is missing", r.app.Spec.EnvName)
		if setClowdStatusErr := SetClowdAppConditions(r.ctx, r.client, r.app, crd.ReconciliationFailed, r.oldStatus, getEnvErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{}, getEnvErr
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) isNamespaceDeleted(namespace string, message string) (ctrl.Result, error) {
	ns := &core.Namespace{}
	if getNSErr := r.client.Get(r.ctx, types.NamespacedName{Name: namespace}, ns); getNSErr != nil {
		return ctrl.Result{Requeue: true}, getNSErr
	}

	if ns.ObjectMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, NewSkippedError(message)
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) isEnvNamespaceDeleted() (ctrl.Result, error) {
	return r.isNamespaceDeleted(r.env.Status.TargetNamespace, "Env target namespace is to be deleted - skipping reconcile")
}

func (r *ClowdAppReconciliation) isAppNamespaceDeleted() (ctrl.Result, error) {
	return r.isNamespaceDeleted(r.app.Namespace, "App namespace is to be deleted - skipping reconcile")
}

func (r *ClowdAppReconciliation) isClowdEnvReconciled() (ctrl.Result, error) {
	if r.env.Generation != r.env.Status.Generation {
		r.recorder.Eventf(r.app, "Warning", "ClowdEnvNotReconciled", "Clowder Environment [%s] is not reconciled", r.app.Spec.EnvName)
		if setClowdStatusErr := SetClowdAppConditions(r.ctx, r.client, r.app, crd.ReconciliationFailed, r.oldStatus, fmt.Errorf("clowd env not reconciled")); setClowdStatusErr != nil {
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{}, NewSkippedError("env not yet reconciled")
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) isEnvReady() (ctrl.Result, error) {
	if !r.env.IsReady() {
		r.recorder.Eventf(r.app, "Warning", "ClowdEnvNotReady", "Clowder Environment [%s] is not ready", r.app.Spec.EnvName)
		if setClowdStatusErr := SetClowdAppConditions(r.ctx, r.client, r.app, crd.ReconciliationFailed, r.oldStatus, fmt.Errorf("clowd env not ready")); setClowdStatusErr != nil {
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		return ctrl.Result{Requeue: true}, NewSkippedError("env isn't ready")
	}
	return ctrl.Result{}, nil
}

var applyOrder = []string{
	"*",
	"Service",
	"Secret",
	"Deployment",
	"Job",
	"CronJob",
	"ScaledObject",
}

func (r *ClowdAppReconciliation) createCache() (ctrl.Result, error) {
	cacheConfig := rc.NewCacheConfig(Scheme, nil, ProtectedGVKs, rc.Options{StrictGVK: true, DebugOptions: DebugOptions, Ordering: applyOrder})
	cache := rc.NewObjectCache(r.ctx, r.client, r.log, cacheConfig)
	r.cache = &cache
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) runProviders() (ctrl.Result, error) {

	r.hashCache.RemoveClowdObjectFromObjects(r.app)

	provider := providers.Provider{
		Client:    r.client,
		Ctx:       r.ctx,
		Env:       r.env,
		Cache:     r.cache,
		Log:       *r.log,
		Config:    r.config,
		HashCache: r.hashCache,
	}

	if provErr := r.runProvidersImplementation(&provider); provErr != nil {
		r.recorder.Eventf(r.app, "Warning", "FailedReconciliation", "Clowdapp requeued [%s]", r.app.GetClowdName())
		if setClowdStatusErr := SetClowdAppConditions(r.ctx, r.client, r.app, crd.ReconciliationFailed, r.oldStatus, provErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		r.log.Info("Provider error", "err", provErr)
		return ctrl.Result{Requeue: true}, provErr
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) runProvidersImplementation(provider *providers.Provider) error {
	// Update app metadata
	updateMetadata(r.app, r.config)

	for _, provAcc := range providers.ProvidersRegistration.Registry {
		provutils.DebugLog(*r.log, "running provider:", "name", provAcc.Name, "order", provAcc.Order)
		prov, err := provAcc.SetupProvider(provider)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("getprov: %s", provAcc.Name), err)
		}
		start := time.Now()
		err = prov.Provide(r.app)
		elapsed := time.Since(start).Seconds()
		providerMetrics.With(prometheus.Labels{"provider": provAcc.Name, "source": "clowdapp"}).Observe(elapsed)
		if err != nil {
			reterr := errors.Wrap(fmt.Sprintf("runapp: %s", provAcc.Name), err)
			reterr.Requeue = true
			return reterr
		}
		provutils.DebugLog(*r.log, "running provider: complete", "name", provAcc.Name, "order", provAcc.Order, "elapsed", fmt.Sprintf("%f", elapsed))
	}

	return nil
}

func (r *ClowdAppReconciliation) applyCache() (ctrl.Result, error) {

	cacheErr := r.cache.ApplyAll()

	if cacheErr != nil {
		r.recorder.Eventf(r.app, "Warning", "FailedReconciliation", "Clowdapp requeued [%s]", r.app.GetClowdName())
		if setClowdStatusErr := SetClowdAppConditions(r.ctx, r.client, r.app, crd.ReconciliationFailed, r.oldStatus, cacheErr); setClowdStatusErr != nil {
			r.log.Info("Set status error", "err", setClowdStatusErr)
			return ctrl.Result{Requeue: true}, setClowdStatusErr
		}
		r.log.Info("Cache error", "err", cacheErr)
		return ctrl.Result{Requeue: true}, cacheErr
	}

	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) setAppResourceStatus() (ctrl.Result, error) {
	if statusErr := SetAppResourceStatus(r.ctx, r.client, r.app); statusErr != nil {
		r.log.Info("Set status error", "err", statusErr)
		return ctrl.Result{Requeue: true}, statusErr
	}
	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) deletedUnusedResources() (ctrl.Result, error) {
	opts := []client.ListOption{
		client.MatchingLabels{r.app.GetPrimaryLabel(): r.app.GetClowdName()},
		client.InNamespace(r.app.Namespace),
	}

	// Delete all resources that are not used anymore
	rErr := r.cache.Reconcile(r.app.GetUID(), opts...)
	if rErr != nil {
		return ctrl.Result{Requeue: true}, NewSkippedError(fmt.Sprintf("error running object cache reconcile: %s", rErr.Error()))
	}

	return ctrl.Result{}, nil
}

func (r *ClowdAppReconciliation) setReconciliationSuccessful() (ctrl.Result, error) {
	if err := ReportDependencies(r.ctx, r.client, r.app, r.env); err != nil {
		r.log.Info("Dependency reporting error", "err", err)
	}

	if setClowdStatusErr := SetClowdAppConditions(r.ctx, r.client, r.app, crd.ReconciliationSuccessful, r.oldStatus, nil); setClowdStatusErr != nil {
		r.log.Info("Set status error", "err", setClowdStatusErr)
		return ctrl.Result{Requeue: true}, setClowdStatusErr
	}
	managedApps[r.app.GetIdent()] = true

	r.recorder.Eventf(r.app, "Normal", "SuccessfulReconciliation", "Clowdapp reconciled [%s]", r.app.GetClowdName())
	r.log.Info("Reconciliation successful")
	return ctrl.Result{}, nil
}
