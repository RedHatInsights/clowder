package controllers

import (
	"context"
	"time"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ClowdAppReferenceReconciler struct {
	Client   client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	ctx      context.Context
}

func (r *ClowdAppReferenceReconciler) Reconcile() (ctrl.Result, error) {
	r.Log.Info("*********************NEW ClowdAppReferenceReconciler")

	return ctrl.Result{}, nil
}

// SetupWithManager registers the CJI with the main manager process
func (r *ClowdAppReferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("clowdappreference")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdAppReference{}).
		Owns(&batchv1.Job{}).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Duration(500*time.Millisecond), time.Duration(60*time.Second)),
		}).
		Complete(r)
}
