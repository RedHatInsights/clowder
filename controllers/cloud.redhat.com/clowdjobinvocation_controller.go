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
	"encoding/json"
	"fmt"
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	deployProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"
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

type ClowdJobWorker struct {
	Cji   *crd.ClowdJobInvocation
	App   *crd.ClowdApp
	Env   *crd.ClowdEnvironment
	Cache *providers.ObjectCache
}

var IqeClowdJob = providers.NewSingleResourceIdent("iqeclowdjob", "core_iqeclowdjob", &batchv1.Job{})
var ClowdJob = providers.NewMultiResourceIdent("clowdjob", "core_clowdjob", &crd.ClowdJobInvocation{})

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
		r.Log.Error(err, "CJI not found ")
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
			r.Recorder.Eventf(&app, "Warning", "JobNameMissing", "ClowdApp [%s] has no job named", cji.Spec.AppName, jobName)
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
			r.Log.Info("Job Invocation Failed", "jobinvocation", jobName, "namespace", app.Namespace)
			r.Recorder.Eventf(&cji, "Warning", "JobNotInvoked", "Job [%s] could not be invoked", jobName)
			return ctrl.Result{Requeue: true}, err
		}
	}

	var emptyIqe crd.IqeJobSpec
	if cji.Spec.Iqe != emptyIqe {

		nn := types.NamespacedName{
			Name:      fmt.Sprintf("%v-%v", app.Name, "iqe"),
			Namespace: app.Namespace,
		}

		j := batchv1.Job{}

		if err := r.createIqeJobResource(&cache, &cji, &env, &app, nn, ctx, &j); err != nil {
			r.Log.Info("Iqe Job creation encountered an error", "jobinvocation", nn.Name, err)
			return ctrl.Result{Requeue: true}, err
		}
		if err := cache.Create(IqeClowdJob, nn, &j); err != nil {
			r.Log.Info("Iqe Job encountered an error", "jobinvocation", nn.Name, err)
			return ctrl.Result{Requeue: true}, err
		}

		cji.Status.Jobs = append(cji.Status.Jobs, j.ObjectMeta.Name)
		r.Log.Info("Iqe Job Invoked Successfully", "jobinvocation", nn.Name, "namespace", app.Namespace)
		r.Recorder.Eventf(&cji, "Normal", "ClowdJobInvoked", "Job [%s] was invoked successfully", j.ObjectMeta.Name)
	}

	// Short running jobs may be done by the time the loop is ranged,
	// so we update again before the reconcile ends

	if statusErr := r.setCompletedStatus(ctx, &cji); statusErr != nil {
		return ctrl.Result{Requeue: true}, statusErr
	}
	if cacheErr := cache.ApplyAll(); cacheErr != nil {
		return ctrl.Result{}, cacheErr
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
		Namespace: app.Namespace,
	}

	j := batchv1.Job{}
	jobProvider.CreateJobResource(cji, env, nn, job, &j)
	if err := cache.Create(ClowdJob, nn, &j); err != nil {
		return err
	}

	cji.Status.Jobs = append(cji.Status.Jobs, j.ObjectMeta.Name)
	r.Log.Info("Job Invoked Successfully", "jobinvocation", job.Name, "namespace", app.Namespace)
	r.Recorder.Eventf(cji, "Normal", "ClowdJobInvoked", "Job [%s] was invoked successfully", j.ObjectMeta.Name)

	return nil
}

func (r *ClowdJobInvocationReconciler) fetchConfig(name types.NamespacedName, ctx context.Context) (config.AppConfig, error) {

	secretConfig := core.Secret{}
	cfg := config.AppConfig{}

	if err := r.Client.Get(ctx, name, &secretConfig); err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(secretConfig.Data["cdappconfig.json"], &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (r *ClowdJobInvocationReconciler) createAndApplyIqeSecret(cache *providers.ObjectCache, ctx context.Context, cji *crd.ClowdJobInvocation, envName string) error {
	iqeSecret := &core.Secret{}

	appList := crd.ClowdAppList{}
	r.Client.List(ctx, &appList)

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-iqe", cji.Spec.AppName),
		Namespace: cji.Namespace,
	}

	var CoreIqeSecret = providers.NewSingleResourceIdent("IqeSecret", "core_iqe_secret", &core.Secret{})

	if err := cache.Create(CoreIqeSecret, nn, iqeSecret); err != nil {
		r.Log.Error(err, "Failed to check for iqe secret")
		return err
	}
	iqeSecret.SetName(nn.Name)
	iqeSecret.SetNamespace(nn.Namespace)

	// This should maybe be owned by the job
	iqeSecret.SetOwnerReferences([]metav1.OwnerReference{cji.MakeOwnerReference()})

	// loop through secrets and get their appConfig
	envConfig := make(map[string]interface{})
	// because we want a list of appConfigs, we need to nest this under the envConfig
	appConfigs := make(map[string]config.AppConfig)
	for _, app := range appList.Items {
		if app.Spec.EnvName == envName {
			appConfig, err := r.fetchConfig(types.NamespacedName{
				Name:      app.Name,
				Namespace: app.Namespace,
			}, ctx)
			if err != nil {
				r.Log.Error(err, "Failed to fetch app config for %v", app)
				return err

			}
			appConfigs[app.Name] = appConfig
		}
	}
	envConfig["cdappconfigs"] = appConfigs

	// Marshall the data into the top level "cdenvconfig.json" to be mounted as a single secret
	// with the appconfigs list embedded
	envData, err := json.Marshal(envConfig)
	if err != nil {
		r.Log.Error(err, "Failed to marshal iqe secret")
		return err
	}

	// and finally cast all these configs and create the secret
	cdEnv := make(map[string][]byte)
	cdEnv["cdenvconfig.json"] = envData
	iqeSecret.Data = cdEnv
	if err := cache.Update(CoreIqeSecret, iqeSecret); err != nil {
		r.Log.Error(err, "Failed to check for iqe secret")
		return err
	}

	return nil
}

func (r *ClowdJobInvocationReconciler) createIqeJobResource(cache *providers.ObjectCache, cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment, app *crd.ClowdApp, nn types.NamespacedName, ctx context.Context, j *batchv1.Job) error {
	labels := cji.GetLabels()
	cji.SetObjectMeta(j, crd.Name(nn.Name), crd.Labels(labels))

	j.ObjectMeta.Labels = labels
	j.Spec.Template.ObjectMeta.Labels = labels

	j.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever

	j.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
		{Name: "quay-cloudservices-pull"},
	}

	pod := crd.PodSpec{
		Resources: env.Spec.Providers.Iqe.Resources,
	}

	envvar := []core.EnvVar{}
	envvar = append(envvar, core.EnvVar{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"})
	envvar = append(envvar, core.EnvVar{Name: "ENV_FOR_DYNACONF", Value: cji.Spec.Iqe.DynaconfEnvName})
	envvar = append(envvar, core.EnvVar{Name: "NAMESPACE", Value: nn.Namespace})
	envvar = append(envvar, core.EnvVar{Name: "CLOWDER_ENABLED", Value: "true"})
	// envvar = append(envvar, core.EnvVar{Name: "IQE_TESTS_LOCAL_CONF_PATH", Value: "/iqe_settings"})

	tag := ""
	if cji.Spec.Iqe.ImageTag != "" {
		tag = cji.Spec.Iqe.ImageTag
	} else {
		tag = app.Spec.Iqe.Plugin
	}
	plugin := app.Spec.Iqe.Plugin
	iqeImage := env.Spec.Providers.Iqe.ImageBase

	accessLevel := env.Spec.Providers.Iqe.K8SAccessLevel

	// TODO: implement access level with service account
	switch accessLevel {
	// Use edit level service account to create and delete resources
	// one per app when the app is created
	case "edit":
		// TODO: Create secret here
		j.Spec.Template.Spec.ServiceAccountName = fmt.Sprintf("%s-iqe", app.Name)
	// Standard view access to the owned resources
	case "view":
		j.Spec.Template.Spec.ServiceAccountName = app.Name
	}

	// var constructedIqeCommand []string
	constructedIqeCommand, err := constructIqeCommand(cji, plugin)
	if err != nil {
		return errors.New("IQE Marker must be set")
	}

	c := core.Container{
		Name:            fmt.Sprintf("%s-iqe", plugin),
		Image:           fmt.Sprintf("%s:%s", iqeImage, tag),
		Command:         constructedIqeCommand,
		Env:             envvar,
		Resources:       deployProvider.ProcessResources(&pod, env),
		VolumeMounts:    []core.VolumeMount{},
		ImagePullPolicy: core.PullIfNotPresent,
	}

	j.Spec.Template.Spec.Volumes = []core.Volume{}
	configAccess := env.Spec.Providers.Iqe.ConfigAccess

	switch configAccess {
	// Build cdenvconfig.json and mount it
	case "environment":
		if secretErr := r.createAndApplyIqeSecret(cache, ctx, cji, env.Name); secretErr != nil {
			r.Log.Error(secretErr, "Cannot apply iqe secret")
		}
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "cdenvconfig",
			MountPath: "/cdenv",
		})

		j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
			Name: "cdenvconfig",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-iqe", cji.Spec.AppName),
				},
			},
		})
		// if we have env access, we also want app access as well, so also run the next case
		fallthrough

	// mount cdappconfig
	case "app":
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "config-secret",
			MountPath: "/cdapp",
		})

		j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
			Name: "config-secret",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: cji.Spec.AppName,
				},
			},
		})

	case "none":
		r.Log.Info("No config mounted to the iqe pod")
	}
	// c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
	// 	Name:      "iqe-settings",
	// 	MountPath: "/iqe_settings",
	// })

	// j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
	// 	Name: "iqe-settings",
	// 	VolumeSource: core.VolumeSource{
	// 		Secret: &core.SecretVolumeSource{
	// 			SecretName: "iqe-settings",
	// 		},
	// 	},
	// })

	j.Spec.Template.Spec.Containers = []core.Container{c}

	return nil
}

func constructIqeCommand(cji *crd.ClowdJobInvocation, plugin string) ([]string, error) {
	command := []string{
		"iqe", "tests", "plugin",
		fmt.Sprintf("%v", strings.ReplaceAll(plugin, "-", "_")),
		"-m", cji.Spec.Iqe.Marker,
	}
	if cji.Spec.Iqe.Filter != "" {
		// Note: go can append multiple values to a slice
		command = append(command, "-k", cji.Spec.Iqe.Filter)
	}
	return command, nil
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
	jobsFinished := getJobStatus(&jobs, cji)
	iqeFinished := getIqeStatus(&jobs, cji)

	cji.Status.Completed = jobsFinished && iqeFinished

	return nil

}

func getJobStatus(jobs *batchv1.JobList, cji *crd.ClowdJobInvocation) bool {
	completionsNeeded := len(cji.Spec.Jobs)
	jobsCompleted := 0

	// if there are no jobs in the spec, the jobs are finished
	if completionsNeeded == 0 {
		return true
	}

	// if there are no jobs run yet, initalize to []string instead of nil
	if len(cji.Status.Jobs) == 0 {
		cji.Status.Jobs = []string{}
		return false
	}
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
	return jobsCompleted == completionsNeeded
}

func getIqeStatus(jobs *batchv1.JobList, cji *crd.ClowdJobInvocation) bool {
	jobCompleted := false
	if len(cji.Status.Jobs) == 0 {
		cji.Status.Jobs = []string{}
		return false
	}

	var emptyIqe crd.IqeJobSpec
	if cji.Spec.Iqe != emptyIqe {
		for _, j := range jobs.Items {
			if contains(cji.Status.Jobs, j.ObjectMeta.Name) {
				if len(j.Status.Conditions) > 0 {
					condition := j.Status.Conditions[0].Type
					if condition == "Complete" || condition == "Failed" {
						jobCompleted = true
					}
				}
			}
		}

	}
	return jobCompleted
}
