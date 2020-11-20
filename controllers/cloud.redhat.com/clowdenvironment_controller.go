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
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/database"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/inmemorydb"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/kafka"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/logging"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/objectstore"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
)

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

//Reconcile fn
func (r *ClowdEnvironmentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("env", req.Name)
	ctx := context.WithValue(context.Background(), errors.ClowdKey("log"), &log)
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

	if env.Status.TargetNamespace == "" {
		if env.Spec.TargetNamespace != "" {
			env.Status.TargetNamespace = env.Spec.TargetNamespace
		} else {
			env.Status.TargetNamespace = env.GenerateTargetNamespace()
			namespace := &core.Namespace{}
			namespace.SetName(env.Status.TargetNamespace)
			err := r.Client.Create(ctx, namespace)
			if err != nil {
				return ctrl.Result{Requeue: true}, err
			}
		}
		err := r.Client.Status().Update(ctx, &env)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &env)

	provider := providers.Provider{
		Ctx:    ctx,
		Client: r.Client,
		Env:    &env,
	}

	err = runProvidersForEnv(provider)

	if err == nil {
		r.Log.Info("Reconciliation successful", "env", env.Name)
		if _, ok := managedEnvironments[env.Name]; ok == false {
			managedEnvironments[env.Name] = true
		}
		managedEnvsMetric.Set(float64(len(managedEnvironments)))
	}

	requeue := errors.HandleError(ctx, err)
	if requeue {
		r.Log.Error(err, "Requeueing due to error")
	}

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

	return ctrl.Result{Requeue: requeue}, nil
}

func runProvidersForEnv(provider providers.Provider) error {
	if err := objectstore.RunEnvProvider(provider); err != nil {
		return errors.Wrap("setupenv: getobjectstore", err)
	}
	if err := logging.RunEnvProvider(provider); err != nil {
		return errors.Wrap("setupenv: logging", err)
	}
	if err := kafka.RunEnvProvider(provider); err != nil {
		return errors.Wrap("setupenv: kafka", err)
	}
	if err := inmemorydb.RunEnvProvider(provider); err != nil {
		return errors.Wrap("setupenv: inmemorydb", err)
	}
	if err := database.RunEnvProvider(provider); err != nil {
		return errors.Wrap("setupenv: database", err)
	}

	return nil
}

// SetupWithManager sets up with manager
func (r *ClowdEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("env")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdEnvironment{}).
		Complete(r)
}

func (r *ClowdEnvironmentReconciler) finalizeEnvironment(reqLogger logr.Logger, e *crd.ClowdEnvironment) error {

	if _, ok := managedEnvironments[e.Name]; ok == true {
		delete(managedEnvironments, e.Name)
	}
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
