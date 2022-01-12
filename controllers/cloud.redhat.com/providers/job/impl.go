package job

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"

	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"

	"k8s.io/apimachinery/pkg/types"
)

// applyJob build the k8s job resource and applies it from the Job config
// defined in the ClowdApp
func CreateJobResource(cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment, app *crd.ClowdApp, nn types.NamespacedName, job *crd.Job, j *batchv1.Job) error {
	labels := cji.GetLabels()
	cji.SetObjectMeta(j, crd.Name(nn.Name), crd.Labels(labels))

	j.ObjectMeta.Labels = labels
	j.ObjectMeta.Labels["job"] = job.Name
	j.Spec.Template.ObjectMeta.Labels = labels
	j.Spec.ActiveDeadlineSeconds = job.ActiveDeadlineSeconds

	pod := job.PodSpec

	if job.RestartPolicy == "" {
		j.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
	} else {
		j.Spec.Template.Spec.RestartPolicy = job.RestartPolicy
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
		Resources:    deployProvider.ProcessResources(&pod, env),
		VolumeMounts: pod.VolumeMounts,
		Ports: []core.ContainerPort{{
			Name:          "metrics",
			ContainerPort: env.Spec.Providers.Metrics.Port,
			Protocol:      core.ProtocolTCP,
		}},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
	}

	if !env.Spec.Providers.Deployment.OmitPullPolicy {
		c.ImagePullPolicy = core.PullIfNotPresent
	}

	if (core.Probe{}) != livenessProbe {
		c.LivenessProbe = &livenessProbe
	}
	if (core.Probe{}) != readinessProbe {
		c.ReadinessProbe = &readinessProbe
	}

	j.Spec.Template.Spec.ServiceAccountName = app.GetClowdSAName()

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "config-secret",
		MountPath: "/cdapp/",
	})

	j.Spec.Template.Spec.Containers = []core.Container{c}

	ics, err := deployProvider.ProcessInitContainers(nn, &c, pod.InitContainers)

	if err != nil {
		return err
	}

	j.Spec.Template.Spec.InitContainers = ics

	j.Spec.Template.Spec.Volumes = pod.Volumes
	j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName: cji.Spec.AppName,
			},
		},
	})
	return nil
}
