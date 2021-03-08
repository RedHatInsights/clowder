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
	maker "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/makers"
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
	err := r.Client.Get(ctx, req.NamespacedName, &cji)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "CJI not found ")
		return ctrl.Result{}, err
	}

	r.Log.Info("Reconciliation started", "ClowdJobInvocation", fmt.Sprintf("%s:%s", cji.Namespace, cji.Name))
	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &cji)

	// Get the ClowdApp. Used to find definition of job being invoked
	app := crd.ClowdApp{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name:      cji.Spec.AppName,
		Namespace: req.Namespace,
	}, &app)

	// Determine if the ClowdApp containing the Job exists
	if err != nil {
		r.Recorder.Eventf(&cji, "Warning", "ClowdAppMissing", "ClowdApp [%s] is missing; Job cannot be invoked", cji.Spec.AppName)
		return ctrl.Result{Requeue: true}, err
	}

	// Determine if the ClowdApp containing the Job is ready
	if app.Status.Ready == false {
		r.Recorder.Eventf(&app, "Warning", "ClowdAppNotReady", "ClowdApp [%s] is not ready", cji.Spec.AppName)
		r.Log.Info("App not yet ready, requeue", "jobinvocation", cji.Spec.AppName, "namespace", app.Namespace)
		return ctrl.Result{Requeue: true}, err
	}

	// Get the ClowdEnv for InvokeJob. Env is needed to build out our pod
	// template for each job
	env := crd.ClowdEnvironment{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name: app.Spec.EnvName,
	}, &env)

	if err != nil {
		r.Recorder.Eventf(&cji, "Warning", "ClowdEnvMissing", "ClowdEnv [%s] is missing; Job cannot be invoked", app.Spec.EnvName)
		return ctrl.Result{Requeue: true}, err
	}

	// Set the initial status to an empty list of pods and a Completed
	// status of false. If a job has been invoked, but hasn't finished,
	// setting the status after requeue will ensure it won't be double
	// invoked
	r.setCompletedStatus(ctx, &cji)
	err = r.Client.Status().Update(ctx, &cji)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	// If the status is updated to complete, don't invoke again.
	if cji.Status.Completed {
		r.Log.Info("Requested jobs are completed", "jobinvocation", cji.Name, "namespace", app.Namespace)
		for _, i := range cji.Status.PodNames {
			r.Log.Info(i, "ran via jobinvocation", cji.Name, "namespace", app.Namespace)
		}
		return ctrl.Result{}, nil
	}

	// Walk the job names to be invoked and match in the ClowdApp Spec
	for _, jobName := range cji.Spec.Jobs {
		// Match the crd.Job name to the JobTemplate in ClowdApp
		jobContent := crd.Job{}
		err := getJobFromName(jobName, &app, &jobContent)
		if err != nil {
			r.Recorder.Eventf(&app, "Error", "JobNameMissing", "ClowdApp [%s] has no job named", cji.Spec.AppName, jobName)
			r.Log.Info("Missing Job Definition", "jobinvocation", cji.Spec.AppName, "namespace", app.Namespace)
			return ctrl.Result{}, err
		}

		fullJobName := fmt.Sprintf("%v-%v-%v", app.Name, jobContent.Name, cji.Name)

		// becuase a CJI can contain > 1 job, we must handle the case
		// where one job is done and the other is still running
		if contains(cji.Status.PodNames, fullJobName) {
			r.Log.Info("Job has already been invoked", "jobinvocation", fullJobName, "namespace", app.Namespace)
			return ctrl.Result{}, nil
		}

		// We have a match that isn't running and can invoke the job
		r.Log.Info("Invoking job", "jobinvocation", jobName, "namespace", app.Namespace)

		if err := r.InvokeJob(ctx, &jobContent, &app, &env, &cji); err != nil {
			r.Log.Info("Job Invocation Failed", "jobinvocation", jobName, "namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, err
		}
	}

	r.setCompletedStatus(ctx, &cji)
	err = r.Client.Status().Update(ctx, &cji)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// InvokeJob is responsible for applying the Job. It also updates and reports
// the status of that job
func (r *ClowdJobInvocationReconciler) InvokeJob(ctx context.Context, job *crd.Job, app *crd.ClowdApp, env *crd.ClowdEnvironment, cji *crd.ClowdJobInvocation) error {
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v-%v", app.Name, job.Name, cji.Name),
		Namespace: app.Namespace,
	}
	j := batchv1.Job{}

	jobType := job.Type
	switch jobType {

	case "request":
		createJobResource(app, env, nn, job, &j)
		if err := r.Client.Create(ctx, &j); err != nil {
			return err
		}
		cji.Status.PodNames = append(cji.Status.PodNames, j.ObjectMeta.Labels["pod"])
		r.Log.Info("Job Invoked Successfully", "jobinvocation", job.Name, "namespace", app.Namespace)

	case "deploy":
		r.Recorder.Eventf(app, "Warning", "Found a deploy type job. To Invoke a Job, ensure the type is set to request in the ClowdApp", "%s is not type request", job.Name)
		r.Log.Info("Job of type Deploy found; skipping", "jobinvocation", job.Name, "namespace", app.Namespace)

	default:
		r.Recorder.Eventf(app, "Warning", "ClowdJobInvocationError", "ClowdJobInvocation [%s] has no type; Job cannot be invoked", app.Name)
		r.Log.Info("Job has no type", "jobinvocation", job.Name, "namespace", app.Namespace)
		return errors.New(fmt.Sprintf("Job has no type %s", job.Name))
	}

	return nil
}

// applyJob build the k8s job resource and applies it from the Job config
// defined in the ClowdApp
func createJobResource(app *crd.ClowdApp, env *crd.ClowdEnvironment, nn types.NamespacedName, job *crd.Job, j *batchv1.Job) error {
	labels := app.GetLabels()
	labels["pod"] = nn.Name
	app.SetObjectMeta(j, crd.Name(nn.Name), crd.Labels(labels))

	pod := job.PodSpec

	j.ObjectMeta.Labels = labels
	j.Spec.Template.ObjectMeta.Labels = labels

	if job.RestartPolicy == "" {
		j.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
	} else {
		j.Spec.Template.Spec.RestartPolicy = job.RestartPolicy
	}

	j.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
		{Name: "quay-cloudservices-pull"},
	}

	envvar := pod.Env
	envvar = append(envvar, core.EnvVar{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"})

	var livenessProbe core.Probe
	var readinessProbe core.Probe

	if pod.LivenessProbe != nil {
		livenessProbe = *pod.LivenessProbe
	} else {
		livenessProbe = core.Probe{}
	}
	if pod.ReadinessProbe != nil {
		readinessProbe = *pod.ReadinessProbe
	} else {
		readinessProbe = core.Probe{}
	}

	c := core.Container{
		Name:         nn.Name,
		Image:        pod.Image,
		Command:      pod.Command,
		Args:         pod.Args,
		Env:          envvar,
		Resources:    maker.ProcessResources(&pod, env),
		VolumeMounts: pod.VolumeMounts,
		Ports: []core.ContainerPort{{
			Name:          "metrics",
			ContainerPort: env.Spec.Providers.Metrics.Port,
		}},
		ImagePullPolicy: core.PullIfNotPresent,
	}

	if (core.Probe{}) != livenessProbe {
		c.LivenessProbe = &livenessProbe
	}
	if (core.Probe{}) != readinessProbe {
		c.ReadinessProbe = &readinessProbe
	}

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "config-secret",
		MountPath: "/cdapp/",
	})

	j.Spec.Template.Spec.Containers = []core.Container{c}

	j.Spec.Template.Spec.InitContainers = maker.ProcessInitContainers(nn, &c, pod.InitContainers)

	j.Spec.Template.Spec.Volumes = pod.Volumes
	j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName: app.ObjectMeta.Name,
			},
		},
	})

	maker.ApplyPodAntiAffinity(&j.Spec.Template)

	annotations := make(map[string]string)
	// Do we need the hash here?
	//annotations["configHash"] = hash
	j.Spec.Template.SetAnnotations(annotations)

	return nil
}

// getJobFromName matches a CJI job name to an App's job definition
func getJobFromName(jobName string, app *crd.ClowdApp, job *crd.Job) error {
	for _, j := range app.Spec.Jobs {
		if j.Name == jobName {
			// dig into pointer issues here &j not working
			*job = j
			return nil
		}
	}
	return errors.New(fmt.Sprintf("No such job %s", jobName))

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
	err := r.Client.Get(ctx, obj, &job)

	cjiList := crd.ClowdJobInvocationList{}
	err = r.Client.List(ctx, &cjiList)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return reqs
		}
		r.Log.Error(err, "Failed to fetch ClowdJobInvocation")
		return nil
	}

	for _, cji := range cjiList.Items {
		// job event triggered a reconcile, check our jobs and match
		// to enable a requeue
		if contains(cji.Status.PodNames, job.ObjectMeta.Labels["pod"]) {
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

// setCompletedStatus will determine if a CJI has completed all needed Jobs
func (r *ClowdJobInvocationReconciler) setCompletedStatus(ctx context.Context, cji *crd.ClowdJobInvocation) error {

	if len(cji.Status.PodNames) == 0 {
		cji.Status.PodNames = []string{}
		return nil
	}

	jobs := batchv1.JobList{}
	err := r.Client.List(ctx, &jobs, client.InNamespace(cji.ObjectMeta.Namespace))
	if err != nil {
		return err
	}
	completionsNeeded := len(cji.Spec.Jobs)
	jobsCompleted := 0

	for _, j := range jobs.Items {
		if contains(cji.Status.PodNames, j.ObjectMeta.Labels["pod"]) {
			if j.Status.Succeeded > 0 {
				jobsCompleted++
			}
		}
	}

	cji.Status.Completed = completionsNeeded == jobsCompleted

	return nil

}
