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

	jinv := crd.ClowdJobInvocation{}
	err := r.Client.Get(ctx, req.NamespacedName, &jinv)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("Reconciliation started", "ClowdJobInvocation", fmt.Sprintf("%s:%s", jinv.Namespace, jinv.Name))
	ctx = context.WithValue(ctx, errors.ClowdKey("obj"), &jinv)

	// Get the ClowdApp
	app := crd.ClowdApp{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name:      jinv.Spec.AppName,
		Namespace: req.Namespace,
	}, &app)

	// Determine if the ClowdApp containing the Job exists
	if err != nil {
		r.Recorder.Eventf(&jinv, "Warning", "ClowdAppMissing", "ClowdApp [%s] is missing; Job cannot be invoked", jinv.Spec.AppName)
		return ctrl.Result{Requeue: true}, err
	}

	// Determine if the Clowapp containing the Job is ready
	if app.Status.Ready == false {
		r.Recorder.Eventf(&app, "Warning", "ClowdAppNotReady", "ClowdApp [%s] is not ready", jinv.Spec.AppName)
		r.Log.Info("App not yet ready, requeue", "jobinvocation", jinv.Spec.AppName, "namespace", app.Namespace)
		return ctrl.Result{Requeue: true}, err
	}

	// Get the ClowdEnv for InvokeJob
	env := crd.ClowdEnvironment{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name: app.Spec.EnvName,
	}, &env)

	if err != nil {
		r.Recorder.Eventf(&jinv, "Warning", "ClowdEnvMissing", "ClowdEnv [%s] is missing; Job cannot be invoked", app.Spec.EnvName)
		return ctrl.Result{Requeue: true}, err
	}

	r.setCompletedStatus(ctx, &jinv)
	err = r.Client.Status().Update(ctx, &jinv)
	if err != nil {
		return ctrl.Result{}, err
	}

	// if the Invocation is already run, don't run again
	// May need to handle a scenario where 1 app is complete and another isn't
	if jinv.Status.Completed {
		r.Log.Info("Requested jobs are completed", "jobinvocation", jinv.Name, "namespace", app.Namespace)
		for _, i := range jinv.Status.PodNames {
			r.Log.Info(i, "ran via jobinvocation", jinv.Name, "namespace", app.Namespace)
		}
		return ctrl.Result{}, nil
	}

	// Walk the job names to be invoked and match in the ClowdApp Spec
	for _, jobName := range jinv.Spec.Jobs {
		// Match the crd.Job name to the JobTemplate in ClowdApp
		jobContent := crd.Job{}
		err := getJobFromName(jobName, &app, &jobContent)
		if err != nil {
			r.Recorder.Eventf(&app, "Error", "JobNameMissing", "ClowdApp [%s] has no job named", jinv.Spec.AppName, jobName)
			r.Log.Info("Missing Job Definition", "jobinvocation", jinv.Spec.AppName, "namespace", app.Namespace)
			return ctrl.Result{}, err
		}

		fullJobName := fmt.Sprintf("%v-%v-%v", app.Name, jobContent.Name, jinv.Name)

		if contains(jinv.Status.PodNames, fullJobName) {
			r.Log.Info("Job has already been invoked", "jobinvocation", fullJobName, "namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, nil
		}

		// We have a match that isn't duplicated and can invoke the job
		r.Log.Info("Invoking job", "jobinvocation", jobName, "namespace", app.Namespace)

		if err := r.InvokeJob(ctx, &jobContent, &app, &env, &jinv); err != nil {
			r.Log.Info("Job Invocation Failed", "jobinvocation", jobName, "namespace", app.Namespace)
			return ctrl.Result{Requeue: true}, err
		}
	}

	r.setCompletedStatus(ctx, &jinv)
	err = r.Client.Status().Update(ctx, &jinv)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// InvokeJob is responsible for applying the Job. It also updates and reports
// the status of that job
func (r *ClowdJobInvocationReconciler) InvokeJob(ctx context.Context, job *crd.Job, app *crd.ClowdApp, env *crd.ClowdEnvironment, jinv *crd.ClowdJobInvocation) error {
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v-%v", app.Name, job.Name, jinv.Name),
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
		jinv.Status.PodNames = append(jinv.Status.PodNames, j.ObjectMeta.Labels["pod"])
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
	r.Recorder = mgr.GetEventRecorderFor("jobinvocation")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.ClowdJobInvocation{}).
		Complete(r)
}

// setCompletedStatus will determine if a CJI has completed all needed Jobs
func (r *ClowdJobInvocationReconciler) setCompletedStatus(ctx context.Context, jinv *crd.ClowdJobInvocation) error {

	if len(jinv.Status.PodNames) == 0 {
		jinv.Status.PodNames = []string{}
		return nil
	}

	jobs := batchv1.JobList{}
	err := r.Client.List(ctx, &jobs, client.InNamespace(jinv.ObjectMeta.Namespace))
	if err != nil {
		return err
	}
	completionsNeeded := len(jinv.Spec.Jobs)
	jobsCompleted := 0

	for _, j := range jobs.Items {
		if contains(jinv.Status.PodNames, j.ObjectMeta.Labels["pod"]) {
			if j.Status.Succeeded > 0 {
				jobsCompleted++
			}
		}
	}

	jinv.Status.Completed = completionsNeeded == jobsCompleted

	return nil

}
