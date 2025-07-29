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

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/iqe"
	jobProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/job"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// ClowdJobInvocationReconciler reconciles a ClowdJobInvocation object
type ClowdJobInvocationReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdjobinvocations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdjobinvocations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps,verbs=get;list;watch
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=clowdapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps;services;persistentvolumeclaims;secrets;events;namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;create;update;watch;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch

// Reconcile CJI Resources
func (r *ClowdJobInvocationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	qualifiedName := fmt.Sprintf("%s:%s", req.Namespace, req.Name)
	log := r.Log.WithValues("jobinvocation", qualifiedName)
	ctx = context.WithValue(ctx, errors.ClowdKey("log"), &log)
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

	cacheConfig := rc.NewCacheConfig(Scheme, nil, ProtectedGVKs, rc.Options{StrictGVK: true, DebugOptions: DebugOptions, Ordering: applyOrder})
	cache := rc.NewObjectCache(ctx, r.Client, &log, cacheConfig)
	cache.AddPossibleGVKFromIdent(
		iqe.IqeSecret,
		iqe.VaultSecret,
		iqe.ClowdJob,
		iqe.IqeClowdJob,
	)

	// Deprecated, used to handle any lagging CJIs that would otherwise throw errors
	if cji.Status.Jobs != nil {

		// Warn, and set the CJI not to reconcile again. If you can see "jobs", the cji has been invoked previously
		r.Log.Info("jobinvocation", cji.Name, "Warning: deprecated CJI status; please remove this CJI and reinvoke to reset the Completed status")
		cji.Status.JobMap = map[string]crd.JobConditionState{}
		if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationSuccessful, nil); condErr != nil {
			return ctrl.Result{}, condErr
		}
		return ctrl.Result{}, nil
	}

	// If the status is updated to complete, don't invoke again.
	if cji.Status.Completed {
		r.Log.Info("Job has been completed", "jobinvocation", cji.Name)
		r.Recorder.Eventf(&cji, "Normal", "ClowdJobInvocationComplete", "ClowdJobInvocation [%s] has completed all jobs", cji.Name)
		if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationSuccessful, nil); condErr != nil {
			return ctrl.Result{}, condErr
		}
		return ctrl.Result{}, nil
	}

	// CJI has already invoked a job, we'll update the status. The Job map must have entries
	// because it can exist to update the status without having done any work.
	if len(cji.Status.JobMap) > 0 {
		if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationSuccessful, nil); condErr != nil {
			return ctrl.Result{}, condErr
		}
		return ctrl.Result{}, nil
	}
	// This is a fresh CJI and needs to be invoked the first time unless it's disabled
	if !cji.Spec.Disabled {
		r.Log.Info("Reconciliation started", "ClowdJobInvocation", fmt.Sprintf("%s:%s", cji.Namespace, cji.Name))
		ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &cji)
	}

	// Get the ClowdApp. Used to find definition of job being invoked
	app := crd.ClowdApp{}
	appErr := r.Client.Get(ctx, types.NamespacedName{
		Name:      cji.Spec.AppName,
		Namespace: req.Namespace,
	}, &app)

	// Set the base map here so we can update the status on errors
	cji.Status.JobMap = map[string]crd.JobConditionState{}

	// Determine if the ClowdApp containing the Job exists
	if appErr != nil {
		r.Recorder.Eventf(&cji, "Warning", "ClowdAppMissing", "ClowdApp [%s] is missing; Job cannot be invoked", cji.Spec.AppName)
		r.Log.Error(appErr, "App not found", "ClowdApp", cji.Spec.AppName, "namespace", cji.Namespace)
		if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, appErr); condErr != nil {
			return ctrl.Result{}, condErr
		}
		// requeue with a buffer to let the app come up
		return ctrl.Result{Requeue: true}, appErr
	}

	// Determine if the ClowdApp containing the Job is ready
	notReadyError := r.HandleNotReady(ctx, app, cji)
	if notReadyError != nil {
		return ctrl.Result{Requeue: true}, notReadyError
	}

	// Get the ClowdEnv for InvokeJob. Env is needed to build out our pod
	// template for each job
	env := crd.ClowdEnvironment{}
	envErr := r.Client.Get(ctx, types.NamespacedName{
		Name: app.Spec.EnvName,
	}, &env)

	if envErr != nil {
		r.Recorder.Eventf(&cji, "Warning", "ClowdEnvMissing", "ClowdEnv [%s] is missing; Job cannot be invoked", app.Spec.EnvName)
		if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, envErr); condErr != nil {
			return ctrl.Result{}, condErr
		}
		// requeue with a buffer to let the env come up
		return ctrl.Result{Requeue: true}, envErr
	}

	// Walk the job names to be invoked and match in the ClowdApp Spec
	for _, jobName := range cji.Spec.Jobs {
		// Match the crd.Job name to the JobTemplate in ClowdApp
		job, err := getJobFromName(jobName, &app)
		if err != nil {
			r.Recorder.Eventf(&app, "Warning", "JobNameMissing", "ClowdApp [%s] has no job defined for that name", jobName)
			r.Log.Info("Error finding job", "jobinvocation", cji.Spec.AppName, "namespace", app.Namespace, "err", err)
			if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, err); condErr != nil {
				return ctrl.Result{}, condErr
			}
			return ctrl.Result{}, err
		}
		job.Name = fmt.Sprintf("%s-%s", app.Name, jobName)

		// We have a match that isn't running and can invoke the job
		r.Log.Info("Invoking job", "jobinvocation", job.Name, "namespace", app.Namespace)

		if err := r.InvokeJob(&cache, &job, &app, &env, &cji); err != nil {
			r.Log.Error(err, "Job Invocation Failed", "jobinvocation", jobName, "namespace", app.Namespace)
			if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, err); condErr != nil {
				return ctrl.Result{}, condErr
			}
			r.Recorder.Eventf(&cji, "Warning", "JobNotInvoked", "Job [%s] could not be invoked", jobName)
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Check IQE struct to see if we need to invoke an IQE Job
	// In the future, we'll need to handle other types, but this will suffice since testing only has iqe.
	var emptyTesting crd.IqeJobSpec
	if cji.Spec.Testing.Iqe != emptyTesting {

		nn := types.NamespacedName{
			Name:      cji.GenerateJobName(),
			Namespace: cji.Namespace,
		}

		j := batchv1.Job{}
		if err := cache.Create(iqe.IqeClowdJob, nn, &j); err != nil {
			r.Log.Error(err, "Iqe Job could not be created via cache", "jobinvocation", nn.Name)
			if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, err); condErr != nil {
				return ctrl.Result{}, condErr
			}
			return ctrl.Result{}, err
		}

		if err := iqe.CreateIqeJobResource(ctx, &cache, &cji, &env, &app, nn, &j, r.Log, r.Client); err != nil {
			r.Log.Error(err, "Iqe Job creation encountered an error", "jobinvocation", nn.Name)
			r.Recorder.Eventf(&cji, "Warning", "IQEJobFailure", "Job [%s] failed to invoke", j.ObjectMeta.Name)
			if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, err); condErr != nil {
				return ctrl.Result{}, condErr
			}
			return ctrl.Result{}, err
		}

		if err := cache.Update(iqe.IqeClowdJob, &j); err != nil {
			r.Log.Error(err, "Iqe Job could not update via cache", "jobinvocation", nn.Name)
			if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, err); condErr != nil {
				return ctrl.Result{}, condErr
			}
			return ctrl.Result{}, err

		}
		r.Log.Info("Iqe Job Invoked Successfully", "jobinvocation", nn.Name, "namespace", app.Namespace)
		if cji.Status.JobMap != nil {
			cji.Status.JobMap[nn.Name] = crd.JobInvoked
		} else {
			cji.Status.JobMap = map[string]crd.JobConditionState{nn.Name: crd.JobInvoked}
		}

		r.Recorder.Eventf(&cji, "Normal", "IQEJobInvoked", "Job [%s] was invoked successfully", j.ObjectMeta.Name)
	}

	if cacheErr := cache.ApplyAll(); cacheErr != nil {
		if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationSuccessful, cacheErr); condErr != nil {
			return ctrl.Result{}, condErr
		}
		return ctrl.Result{}, cacheErr
	}

	// Short running jobs may be done by the time the loop is ranged,
	// so we update again before the reconcile ends
	if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationSuccessful, nil); condErr != nil {
		return ctrl.Result{}, condErr
	}

	return ctrl.Result{}, nil
}

// InvokeJob is responsible for applying the Job. It also updates and reports
// the status of that job
func (r *ClowdJobInvocationReconciler) InvokeJob(cache *rc.ObjectCache, job *crd.Job, app *crd.ClowdApp, env *crd.ClowdEnvironment, cji *crd.ClowdJobInvocation) error {
	// Update job name to avoid collisions
	randomString := utils.RandStringLower(7)
	jobName := fmt.Sprintf("%s-%s", job.Name, randomString)

	nn := types.NamespacedName{
		Namespace: cji.Namespace,
	}

	labelMaxLength := 63

	if len(jobName) > labelMaxLength {
		return errors.NewClowderError(fmt.Sprintf("[%s] contains a label with character length greater than 63", jobName))
	}
	nn.Name = jobName

	j := batchv1.Job{}
	if err := cache.Create(iqe.ClowdJob, nn, &j); err != nil {
		return err
	}

	if err := jobProvider.CreateJobResource(cji, env, app, nn, job, &j); err != nil {
		return err
	}

	if err := cache.Update(iqe.ClowdJob, &j); err != nil {
		return err
	}

	r.Log.Info("Job Invoked Successfully", "jobinvocation", job.Name, "namespace", app.Namespace)
	if cji.Status.JobMap != nil {
		cji.Status.JobMap[j.ObjectMeta.Name] = crd.JobInvoked
	} else {
		cji.Status.JobMap = map[string]crd.JobConditionState{j.ObjectMeta.Name: crd.JobInvoked}
	}
	r.Recorder.Eventf(cji, "Normal", "ClowdJobInvoked", "Job [%s] was invoked successfully", j.ObjectMeta.Name)

	return nil
}

// getJobFromName matches a CJI job name to an App's job definition
func getJobFromName(jobName string, app *crd.ClowdApp) (job crd.Job, err error) {
	for _, j := range app.Spec.Jobs {
		if j.Name == jobName {
			if j.Disabled {
				return crd.Job{}, errors.NewClowderError(fmt.Sprintf("Job [%s] is disabled", jobName))
			}
			return j, nil
		}
	}
	return crd.Job{}, errors.NewClowderError(fmt.Sprintf("No such job %s", jobName))
}

// SetupWithManager registers the CJI with the main manager process
func (r *ClowdJobInvocationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("clowdjobinvocation")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdJobInvocation{}).
		Owns(&batchv1.Job{}).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](time.Duration(500*time.Millisecond), time.Duration(60*time.Second)),
		}).
		Complete(r)
}

func UpdateInvokedJobStatus(jobs *batchv1.JobList, cji *crd.ClowdJobInvocation) error {

	for _, s := range jobs.Items {
		jobName := s.ObjectMeta.Name
		if _, ok := cji.Status.JobMap[jobName]; ok {
			for _, c := range s.Status.Conditions {
				condition := c.Type
				switch condition {
				case batchv1.JobComplete:
					cji.Status.JobMap[jobName] = crd.JobComplete
				case batchv1.JobFailed:
					cji.Status.JobMap[jobName] = crd.JobFailed
				default:
					curr := cji.Status.JobMap[jobName]
					if curr != crd.JobComplete && curr != crd.JobFailed {
						cji.Status.JobMap[jobName] = crd.JobInvoked
					}
				}
			}
		}
	}
	return nil
}

func GetJobsStatus(jobs *batchv1.JobList, cji *crd.ClowdJobInvocation) bool {

	jobsRequired := len(cji.Spec.Jobs)
	var emptyTesting crd.IqeJobSpec
	if cji.Spec.Testing.Iqe != emptyTesting {
		jobsRequired++
	}
	jobsCompleted := countCompletedJobs(jobs, cji)
	return jobsCompleted == jobsRequired
}

func countCompletedJobs(jobs *batchv1.JobList, cji *crd.ClowdJobInvocation) int {

	jobsCompleted := 0

	// A job either completes successfully, or fails to succeed within the
	// backoffLimit threshold. The Condition status is only populated when
	// the jobs have succeeded or passed the backoff limit
	for _, s := range jobs.Items {
		jobName := s.ObjectMeta.Name
		if _, ok := cji.Status.JobMap[jobName]; ok {
			for _, c := range s.Status.Conditions {
				condition := c.Type
				if condition == batchv1.JobComplete || condition == batchv1.JobFailed {
					jobsCompleted++
				}
			}
		}
	}
	return jobsCompleted
}

func (r *ClowdJobInvocationReconciler) HandleNotReady(ctx context.Context, app crd.ClowdApp, cji crd.ClowdJobInvocation) error {
	if app.IsReady() {
		return nil
	}
	if cji.Spec.RunOnNotReady {
		return nil
	}
	r.Recorder.Eventf(&app, "Warning", "ClowdAppNotReady", "ClowdApp [%s] is not ready", cji.Spec.AppName)
	r.Log.Info("App not yet ready, requeue", "jobinvocation", cji.Spec.AppName, "namespace", app.Namespace)
	readyErr := errors.NewClowderError(fmt.Sprintf("The %s app must be ready for CJI to start", cji.Spec.AppName))
	if condErr := SetClowdJobInvocationConditions(ctx, r.Client, &cji, crd.ReconciliationFailed, readyErr); condErr != nil {
		return condErr
	}

	// requeue with a buffer to let the app come up
	return readyErr
}
