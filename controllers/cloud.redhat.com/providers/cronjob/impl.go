package cronjob

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetCronJobName(app *crd.ClowdApp, cronjob *crd.Job) string {
	return fmt.Sprintf("%s-%s", app.Name, cronjob.Name)
}

func (j *cronjobProvider) makeCronJob(cronjob *crd.Job, app *crd.ClowdApp) error {

	nn := types.NamespacedName{
		Name:      GetCronJobName(app, cronjob),
		Namespace: app.Namespace,
	}

	pt := core.PodTemplateSpec{}

	if err := buildPodTemplate(app, j.Env, &pt, nn, cronjob); err != nil {
		return err
	}

	c := &batch.CronJob{}
	if err := j.Cache.Create(CoreCronJob, nn, c); err != nil {
		return err
	}

	applyCronJob(app, c, &pt, nn, cronjob)

	return j.Cache.Update(CoreCronJob, c)
}

func buildPodTemplate(app *crd.ClowdApp, env *crd.ClowdEnvironment, pt *core.PodTemplateSpec, nn types.NamespacedName, cronjob *crd.Job) error {
	labels := app.GetLabels()
	labels["pod"] = nn.Name

	pod := cronjob.PodSpec

	pt.Labels = labels

	envvar := pod.Env
	envvar = append(envvar, core.EnvVar{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"})

	for _, env := range envvar {
		if env.ValueFrom != nil {
			if env.ValueFrom.FieldRef != nil {
				if env.ValueFrom.FieldRef.APIVersion == "" {
					env.ValueFrom.FieldRef.APIVersion = "v1"
				}
			}
		}
	}

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
	}

	if !env.Spec.Providers.Deployment.OmitPullPolicy {
		c.ImagePullPolicy = core.PullIfNotPresent
	}

	if pod.MachinePool != "" {
		pt.Spec.Tolerations = []core.Toleration{{
			Key:      pod.MachinePool,
			Effect:   core.TaintEffectNoSchedule,
			Operator: core.TolerationOpEqual,
			Value:    "true",
		}}
	} else {
		pt.Spec.Tolerations = []core.Toleration{}
	}

	// set service account for pod
	pt.Spec.ServiceAccountName = app.GetClowdSAName()
	pt.Spec.TerminationGracePeriodSeconds = utils.Int64Ptr(30)
	pt.Spec.SecurityContext = &core.PodSecurityContext{}
	pt.Spec.SchedulerName = "default-scheduler"
	pt.Spec.DNSPolicy = core.DNSClusterFirst

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

	pt.Spec.Containers = []core.Container{c}

	ics, err := deployProvider.ProcessInitContainers(nn, &c, pod.InitContainers)

	if err != nil {
		return err
	}
	pt.Spec.InitContainers = ics

	pt.Spec.Volumes = pod.Volumes
	pt.Spec.Volumes = append(pt.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				DefaultMode: utils.Int32Ptr(420),
				SecretName:  app.Name,
			},
		},
	})

	for i := range pt.Spec.Volumes {
		vol := &pt.Spec.Volumes[i]
		if vol.ConfigMap != nil && (vol.ConfigMap.DefaultMode == nil || *vol.ConfigMap.DefaultMode == 0) {
			vol.ConfigMap.DefaultMode = utils.Int32Ptr(420)
		} else if vol.Secret != nil && (vol.Secret.DefaultMode == nil || *vol.Secret.DefaultMode == 0) {
			vol.Secret.DefaultMode = utils.Int32Ptr(420)
		}
	}

	deployProvider.ApplyPodAntiAffinity(pt)

	return nil
}

func applyCronJob(app *crd.ClowdApp, cj *batch.CronJob, pt *core.PodTemplateSpec, nn types.NamespacedName, cronjob *crd.Job) {
	labels := app.GetLabels()
	labels["pod"] = nn.Name
	app.SetObjectMeta(cj, crd.Name(nn.Name), crd.Labels(labels))

	utils.UpdateAnnotations(pt, provutils.KubeLinterAnnotations)
	utils.UpdateAnnotations(cj, provutils.KubeLinterAnnotations, app.Annotations)

	cj.Spec.Schedule = cronjob.Schedule

	cj.Spec.JobTemplate.Labels = labels
	cj.Spec.JobTemplate.Spec.Template = *pt
	cj.Spec.JobTemplate.Spec.ActiveDeadlineSeconds = cronjob.ActiveDeadlineSeconds

	if cronjob.ConcurrencyPolicy == "" {
		cj.Spec.ConcurrencyPolicy = batch.AllowConcurrent
	} else {
		cj.Spec.ConcurrencyPolicy = cronjob.ConcurrencyPolicy
	}

	if cronjob.StartingDeadlineSeconds != nil {
		cj.Spec.StartingDeadlineSeconds = cronjob.StartingDeadlineSeconds
	}

	if cronjob.Suspend != nil {
		cj.Spec.Suspend = cronjob.Suspend
	} // implicit else => default is *bool false

	if cronjob.SuccessfulJobsHistoryLimit != nil {
		cj.Spec.SuccessfulJobsHistoryLimit = cronjob.SuccessfulJobsHistoryLimit
	} // implicit else => default is 3

	if cronjob.FailedJobsHistoryLimit != nil {
		cj.Spec.FailedJobsHistoryLimit = cronjob.FailedJobsHistoryLimit
	} // implicit else => default is 1

	if cronjob.RestartPolicy == "" {
		cj.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
	} else {
		cj.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = cronjob.RestartPolicy
	}

	if cronjob.Parallelism != nil {
		cj.Spec.JobTemplate.Spec.Parallelism = cronjob.Parallelism
	} // implicit else => default is 1

	if cronjob.Completions != nil {
		cj.Spec.JobTemplate.Spec.Completions = cronjob.Completions
	} // implicit else => default is 1

	deployProvider.ApplyPodAntiAffinity(&cj.Spec.JobTemplate.Spec.Template)

}
