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
	//proxyClient := ProxyClient{Ctx: ctx, Client: r.Client}
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

	// Get the ClowdApp
	app := crd.ClowdApp{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name:      jinv.Spec.AppName,
		Namespace: req.Namespace,
	}, &app)

	// Determine if the ClowdApp containing the Job exists
	if err != nil {
		r.Recorder.Eventf(&jinv, "Warning", "ClowdAppMissing", "ClowdApp [%s] is missing; Job cannot be jinved", jinv.Spec.AppName)
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

	// Walk the job names to be invoked and match in the ClowdApp Spec
	for _, job := range jinv.Spec.Jobs {
		jobContent, err := matchAndReturnJob(job, &app)
		if err != nil {
			r.Recorder.Eventf(&app, "Error", "JobNameMissing", "ClowdApp [%s] has no job named", jinv.Spec.AppName, job)
			r.Log.Info("Missing Job Definition", "jobinvocation", jinv.Spec.AppName, "namespace", app.Namespace)
			return ctrl.Result{}, err
		}
		r.Log.Info("Invoking job", "jobinvocation", job, "namespace", app.Namespace)
		if err := r.InvokeJob(jobContent, &app, &env, ctx); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *JobInvocationReconciler) InvokeJob(job crd.Job, app *crd.ClowdApp, env *crd.ClowdEnvironment, ctx context.Context) error {
	now := time.Now()
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v-%v", app.Name, job.Name, now.Unix()),
		Namespace: app.Namespace,
	}
	if !job.OnRequest {
		r.Recorder.Eventf(app, "Warning", "Only OnRequest Jobs are supported at this time", "%s is not set to onRequest", job)
		r.Log.Info("Unsupported Job Type", "jobinvocation", job.Name, "namespace", app.Namespace)
	} else {
		j := batchv1.Job{}
		createJob(app, env, nn, job, &j)
		if err := r.Client.Create(ctx, &j); err != nil {
			return err
		}
	}
	return nil
}

func createJob(app *crd.ClowdApp, env *crd.ClowdEnvironment, nn types.NamespacedName, job crd.Job, j *batchv1.Job) error {
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
	//annotations["configHash"] = hash
	j.Spec.Template.SetAnnotations(annotations)

	return nil
}

func matchAndReturnJob(jobName string, app *crd.ClowdApp) (crd.Job, error) {
	for _, j := range app.Spec.Jobs {
		if j.Name == jobName {
			return j, nil
		}
	}
	return crd.Job{}, errors.New(fmt.Sprintf("No such job %s", jobName))

}

func (r *JobInvocationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("jobinvocation")
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.JobInvocation{}).
		Complete(r)
}
