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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &env)

	provider := providers.Provider{
		Ctx:    ctx,
		Client: r.Client,
		Env:    &env,
	}

	err = runProvidersForEnv(provider)

	if err == nil {
		r.Log.Info("Reconciliation successful", "env", env.Name)
	}

	requeue := errors.HandleError(ctx, err)
	if requeue {
		r.Log.Error(err, "Requeueing due to error")
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
