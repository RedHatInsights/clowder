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
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/iqe"
	jobProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/job"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

// ClowdJobInvocationReconciler reconciles a ClowdJobInvocation object
type ClowdJobInvocationReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

var IqeClowdJob = providers.NewSingleResourceIdent("cji", "iqe_clowdjob", &batchv1.Job{})
var ClowdJob = providers.NewMultiResourceIdent("cji", "clowdjob", &batchv1.Job{})
var IqeSecret = providers.NewSingleResourceIdent("cji", "iqe_secret", &core.Secret{})

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdjobinvocations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdjobinvocations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps,verbs=get;list;watch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps;services;persistentvolumeclaims;secrets;events;namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;create;update;watch;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch

// Reconcile CJI Resources
func (r *ClowdJobInvocationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	qualifiedName := fmt.Sprintf("%s:%s", req.Namespace, req.Name)
	log := r.Log.WithValues("jobinvocation", qualifiedName)
	ctx := context.WithValue(context.Background(), errors.ClowdKey("log"), &log)
	ctx = context.WithValue(ctx, errors.ClowdKey("recorder"), &r.Recorder)

	cji := crd.ClowdJobInvocation{}
	if err := r.Client.Get(ctx, req.NamespacedName, &cji); err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "CJI not found")
		return ctrl.Result{}, err
	}

	cache := providers.NewObjectCache(ctx, r.Client, r.Scheme)

	// Set the initial status to an empty list of pods and a Completed
	// status of false. If a job has been invoked, but hasn't finished,
	// setting the status after requeue will ensure it won't be double invoked
	if err := r.setCompletedStatus(ctx, &cji); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	if err := r.Client.Status().Update(ctx, &cji); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	// If the status is updated to complete, don't invoke again.
	if cji.Status.Completed {
		r.Recorder.Eventf(&cji, "Normal", "ClowdJobInvocationComplete", "ClowdJobInvocation [%s] has completed all jobs", cji.Name)
		return ctrl.Result{}, nil
	}

	// We have already invoked jobs and don't need to announce another reconcile run
	if len(cji.Status.Jobs) > 0 {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciliation started", "ClowdJobInvocation", fmt.Sprintf("%s:%s", cji.Namespace, cji.Name))
	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &cji)

	// Get the ClowdApp. Used to find definition of job being invoked
	app := crd.ClowdApp{}
	appErr := r.Client.Get(ctx, types.NamespacedName{
		Name:      cji.Spec.AppName,
		Namespace: req.Namespace,
	}, &app)

	// Determine if the ClowdApp containing the Job exists
	if appErr != nil {
		r.Recorder.Eventf(&cji, "Warning", "ClowdAppMissing", "ClowdApp [%s] is missing; Job cannot be invoked", cji.Spec.AppName)
		return ctrl.Result{Requeue: true}, appErr
	}

	// Determine if the ClowdApp containing the Job is ready
	if !app.IsReady() {
		r.Recorder.Eventf(&app, "Warning", "ClowdAppNotReady", "ClowdApp [%s] is not ready", cji.Spec.AppName)
		r.Log.Info("App not yet ready, requeue", "jobinvocation", cji.Spec.AppName, "namespace", app.Namespace)
		return ctrl.Result{Requeue: true}, appErr
	}

	// Get the ClowdEnv for InvokeJob. Env is needed to build out our pod
	// template for each job
	env := crd.ClowdEnvironment{}
	envErr := r.Client.Get(ctx, types.NamespacedName{
		Name: app.Spec.EnvName,
	}, &env)

	if envErr != nil {
		r.Recorder.Eventf(&cji, "Warning", "ClowdEnvMissing", "ClowdEnv [%s] is missing; Job cannot be invoked", app.Spec.EnvName)
		return ctrl.Result{Requeue: true}, envErr
	}

	// Walk the job names to be invoked and match in the ClowdApp Spec
	for _, jobName := range cji.Spec.Jobs {
		// Match the crd.Job name to the JobTemplate in ClowdApp
		job, err := getJobFromName(jobName, &app)
		if err != nil {
			r.Recorder.Eventf(&app, "Warning", "JobNameMissing", "ClowdApp [%s] has no job named", jobName)
			r.Log.Info("Missing Job Definition", "jobinvocation", cji.Spec.AppName, "namespace", app.Namespace)
			return ctrl.Result{}, err
		}

		// becuase a CJI can contain > 1 job, we must handle the case
		// where one job is done and the other is still running
		fullJobName := fmt.Sprintf("%v-%v-%v", app.Name, job.Name, cji.Name)
		if contains(cji.Status.Jobs, fullJobName) {
			continue
		}

		// We have a match that isn't running and can invoke the job
		r.Log.Info("Invoking job", "jobinvocation", jobName, "namespace", app.Namespace)

		if err := r.InvokeJob(&cache, &job, &app, &env, &cji); err != nil {
			r.Log.Error(err, "Job Invocation Failed", "jobinvocation", jobName, "namespace", app.Namespace)
			r.Recorder.Eventf(&cji, "Warning", "JobNotInvoked", "Job [%s] could not be invoked", jobName)
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Check IQE struct to see if we need to invoke an IQE Job
	// In the future, we'll need to handle other types, but this will suffice since testing only has iqe.
	var emptyTesting crd.IqeJobSpec
	if cji.Spec.Testing.Iqe != emptyTesting {

		nn := types.NamespacedName{
			Name:      fmt.Sprintf("%s-iqe", cji.Name),
			Namespace: cji.Namespace,
		}

		j := batchv1.Job{}
		if err := cache.Create(IqeClowdJob, nn, &j); err != nil {
			r.Log.Error(err, "Iqe Job could not be created via cache", "jobinvocation", nn.Name)
			return ctrl.Result{}, err
		}

		if err := iqe.CreateIqeJobResource(&cache, &cji, &env, &app, nn, ctx, &j, r.Log, r.Client); err != nil {
			r.Log.Error(err, "Iqe Job creation encountered an error", "jobinvocation", nn.Name)
			r.Recorder.Eventf(&cji, "Warning", "IQEJobFailure", "Job [%s] failed to invoke", j.ObjectMeta.Name)
			return ctrl.Result{}, err
		}

		if err := cache.Update(IqeClowdJob, &j); err != nil {
			r.Log.Error(err, "Iqe Job could not update via cache", "jobinvocation", nn.Name)
			return ctrl.Result{}, err

		}
		cji.Status.Jobs = append(cji.Status.Jobs, j.ObjectMeta.Name)
		r.Log.Info("Iqe Job Invoked Successfully", "jobinvocation", nn.Name, "namespace", app.Namespace)
		r.Recorder.Eventf(&cji, "Normal", "IQEJobInvoked", "Job [%s] was invoked successfully", j.ObjectMeta.Name)
	}

	if cacheErr := cache.ApplyAll(); cacheErr != nil {
		return ctrl.Result{}, cacheErr
	}

	// Short running jobs may be done by the time the loop is ranged,
	// so we update again before the reconcile ends
	if statusErr := r.setCompletedStatus(ctx, &cji); statusErr != nil {
		return ctrl.Result{Requeue: true}, statusErr
	}

	if updateErr := r.Client.Status().Update(ctx, &cji); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, nil
}

// InvokeJob is responsible for applying the Job. It also updates and reports
// the status of that job
func (r *ClowdJobInvocationReconciler) InvokeJob(cache *providers.ObjectCache, job *crd.Job, app *crd.ClowdApp, env *crd.ClowdEnvironment, cji *crd.ClowdJobInvocation) error {
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v-%v", app.Name, job.Name, cji.Name),
		Namespace: cji.Namespace,
	}

	j := batchv1.Job{}
	if err := cache.Create(ClowdJob, nn, &j); err != nil {
		return err
	}

	jobProvider.CreateJobResource(cji, env, nn, job, &j)

	if err := cache.Update(ClowdJob, &j); err != nil {
		return err
	}

	cji.Status.Jobs = append(cji.Status.Jobs, j.ObjectMeta.Name)
	r.Log.Info("Job Invoked Successfully", "jobinvocation", job.Name, "namespace", app.Namespace)
	r.Recorder.Eventf(cji, "Normal", "ClowdJobInvoked", "Job [%s] was invoked successfully", j.ObjectMeta.Name)

	return nil
}

// getJobFromName matches a CJI job name to an App's job definition
func getJobFromName(jobName string, app *crd.ClowdApp) (job crd.Job, err error) {
	for _, j := range app.Spec.Jobs {
		if j.Name == jobName {
			return j, nil
		}
	}
	return crd.Job{}, errors.New(fmt.Sprintf("No such job %s", jobName))
}

// SetupWithManager registers the CJI with the main manager process
func (r *ClowdJobInvocationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("clowdjobinvocation")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdJobInvocation{}).
		Watches(
			&source.Kind{Type: &batchv1.Job{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(r.cjiToEnqueueUponJobUpdate)},
		).
		Owns(&batchv1.Job{}).
		Complete(r)
}

// cjiToEnqueueUponJobUpdate watches is triggered when a job watched by the
// ClowdJobInvocationReconciler is updated. Rather than constantly requeue
// in order to update a cji status, we can trigger a queue up a single reconcile
// when a watched job updates
func (r *ClowdJobInvocationReconciler) cjiToEnqueueUponJobUpdate(a handler.MapObject) []reconcile.Request {
	reqs := []reconcile.Request{}
	ctx := context.Background()
	obj := types.NamespacedName{
		Name:      a.Meta.GetName(),
		Namespace: a.Meta.GetNamespace(),
	}

	job := batchv1.Job{}
	if cjErr := r.Client.Get(ctx, obj, &job); cjErr != nil {
		if k8serr.IsNotFound(cjErr) {
			// Must have been deleted
			return reqs
		}
		r.Log.Error(cjErr, "Failed to fetch ClowdJob")
		return nil
	}

	cjiList := crd.ClowdJobInvocationList{}
	if cjiErr := r.Client.List(ctx, &cjiList); cjiErr != nil {
		if k8serr.IsNotFound(cjiErr) {
			// Must have been deleted
			return reqs
		}
		r.Log.Error(cjiErr, "Failed to fetch ClowdJobInvocation")
		return nil
	}

	for _, cji := range cjiList.Items {
		// job event triggered a reconcile, check our jobs and match
		// to enable a requeue
		if contains(cji.Status.Jobs, job.ObjectMeta.Name) {
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      cji.Name,
					Namespace: cji.Namespace,
				},
			})
		}
	}

	return reqs
}

// Look for completed instead of successes
// setCompletedStatus will determine if a CJI has completed all needed Jobs
func (r *ClowdJobInvocationReconciler) setCompletedStatus(ctx context.Context, cji *crd.ClowdJobInvocation) error {

	jobs := batchv1.JobList{}
	if err := r.Client.List(ctx, &jobs, client.InNamespace(cji.ObjectMeta.Namespace)); err != nil {
		return err
	}

	cji.Status.Completed = getInvocationStatus(&jobs, cji)

	return nil
}

func getInvocationStatus(jobs *batchv1.JobList, cji *crd.ClowdJobInvocation) bool {

	// if there are no jobs run yet, initalize to []string instead of nil
	if len(cji.Status.Jobs) == 0 {
		cji.Status.Jobs = []string{}
		return false
	}

	var completed bool
	jobsCompleted := countCompletedJobs(jobs, cji)
	// If calling jobs, we aren't complete until every job has completed
	if invokedJobs := len(cji.Spec.Jobs); invokedJobs > 0 {
		completed = jobsCompleted == invokedJobs
	} else {
		// only one iqe job will ever be invoked at a time, so it's one and done
		completed = jobsCompleted > 0
	}
	return completed
}

func countCompletedJobs(jobs *batchv1.JobList, cji *crd.ClowdJobInvocation) int {

	jobsCompleted := 0

	// A job either completes successfully, or fails to succeed within the
	// backoffLimit threshold. The Condition status is only populated when
	// the jobs have succeeded or passed the backoff limit
	for _, j := range jobs.Items {
		if contains(cji.Status.Jobs, j.ObjectMeta.Name) {
			if len(j.Status.Conditions) > 0 {
				condition := j.Status.Conditions[0].Type
				if condition == "Complete" || condition == "Failed" {
					jobsCompleted++
				}
			}
		}
	}
	return jobsCompleted
}
