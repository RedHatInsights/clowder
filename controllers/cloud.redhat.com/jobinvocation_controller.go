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

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

// JobInvocationReconciler reconciles a JobInvocation object
type JobInvocationReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=jobinvocations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=jobinvocations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps,verbs=get;list;watch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps;services;persistentvolumeclaims;secrets;events;namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;create;update;watch;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch

func (r *JobInvocationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	qualifiedName := fmt.Sprintf("%s:%s", req.Namespace, req.Name)
	log := r.Log.WithValues("jobinvocation", qualifiedName)
	ctx := context.WithValue(context.Background(), errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)
	// proxyClient := ProxyClient{Ctx: ctx, Client: r.Client}
	jinv := crd.JobInvocation{}
	err := r.Client.Get(ctx, req.NamespacedName, &jinv)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	r.Log.Info("Reconciliation started", "JobInvocation", fmt.Sprintf("%s:%s", jinv.Namespace, jinv.Name))
	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &jinv)

	for _, job := range jinv.Spec.Jobs {
		fmt.Printf("Invoking job %s\n", job)
	}

	fmt.Printf("Looking for app name %s\n", jinv.Spec.AppName)
	// Get the ClowdApp
	// We also need to get delayed jobs that are oneshot...
	app := crd.ClowdApp{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name:      jinv.Spec.AppName,
		Namespace: req.Namespace,
	}, &app)
	fmt.Printf("testing app %s\n", app.Name)
	fmt.Printf("app %v\n", app)

	if err != nil {
		r.Recorder.Eventf(&jinv, "Warning", "ClowdAppMissing", "ClowdApp [%s] is missing; Job cannot be jinved", jinv.Spec.AppName)
		return ctrl.Result{Requeue: true}, err
	}

	if app.Status.Ready == true {
		fmt.Print("app is ready")
		r.Recorder.Eventf(&jinv, "Warning", "ClowdAppNotReady", "ClowdApp [%s] is not ready", jinv.Spec.AppName)
		r.Log.Info("App not yet ready", "jobinvocation", jinv.Spec.AppName, "namespace", app.Namespace)
		return ctrl.Result{Requeue: true}, err
	}

	// Iterate through Jobs
	// "Run" the job
	// Skip cronjobs and already run jobs
	for _, job := range jinv.Spec.Jobs {
		fmt.Printf("Invoking job %s\n", job)
	}

	return ctrl.Result{}, nil
}

func (r *JobInvocationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.JobInvocation{}).
		Complete(r)
}
