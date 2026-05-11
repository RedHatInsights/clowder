// Package job provides job and cron job management for Clowder applications
package job

import (
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	"k8s.io/apimachinery/pkg/types"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// CreateJobResource builds the k8s job resource and applies it from the Job config
// defined in the ClowdApp
func CreateJobResource(cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment, app *crd.ClowdApp, nn types.NamespacedName, job *crd.Job, j *batchv1.Job) error {
	labels := cji.GetLabels()
	cji.SetObjectMeta(j, crd.Name(nn.Name), crd.Labels(labels))

	j.Labels = labels
	j.Labels["job"] = job.Name
	j.Spec.Template.Labels = labels
	j.Spec.ActiveDeadlineSeconds = job.ActiveDeadlineSeconds

	pod := job.PodSpec

	if job.RestartPolicy == "" {
		j.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
	} else {
		j.Spec.Template.Spec.RestartPolicy = job.RestartPolicy
	}

	if job.Parallelism != nil {
		j.Spec.Parallelism = job.Parallelism
	}

	if job.Completions != nil {
		j.Spec.Completions = job.Completions
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

	if pod.MachinePool != "" {
		j.Spec.Template.Spec.Tolerations = []core.Toleration{{
			Key:      pod.MachinePool,
			Effect:   core.TaintEffectNoSchedule,
			Operator: core.TolerationOpEqual,
			Value:    "true",
		}}
	} else {
		j.Spec.Template.Spec.Tolerations = []core.Toleration{}
	}

	if !env.Spec.Providers.Deployment.OmitPullPolicy {
		c.ImagePullPolicy = core.PullIfNotPresent
	} else {
		imageComponents := strings.Split(c.Image, ":")
		if len(imageComponents) > 1 {
			if imageComponents[1] == "latest" {
				c.ImagePullPolicy = core.PullAlways
			} else {
				c.ImagePullPolicy = core.PullIfNotPresent
			}
		} else {
			c.ImagePullPolicy = core.PullIfNotPresent
		}
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

	if env.Spec.Providers.Web.TLS.Enabled {
		// Get CA info from app spec
		caSecretName, caFileName := resolveCAForJob(env, app)
		provutils.AddCertVolumeWithCA(&j.Spec.Template.Spec, nn.Name, caSecretName, caFileName)
	}

	utils.UpdateAnnotations(&j.Spec.Template, provutils.KubeLinterAnnotations, cji.Annotations)
	utils.UpdateAnnotations(j, provutils.KubeLinterAnnotations, app.Annotations)

	return nil
}

// resolveCAForJob determines which CA secret to mount based on app's CA configuration
// Returns (secretName, fileName)
// - ("", "service-ca.crt") for default (no CA specified)
// - ("", "") for system-trust-store (skip mounting)
// - ("{env}-ca-bundle", "{caname}.crt") for CA from environment bundle
// - ("{override-secret-name}", "ca.crt") for override secret
func resolveCAForJob(env *crd.ClowdEnvironment, app *crd.ClowdApp) (string, string) {
	// Case 1: App uses override secret
	if app.Spec.TLSCertificateAuthoritySecretRef != nil {
		// Mount the app-managed secret with standard ca.crt key
		return app.Spec.TLSCertificateAuthoritySecretRef.Name, "ca.crt"
	}

	// Case 2: No CA specified - use default
	if app.Spec.TLSCertificateAuthorityName == nil {
		return "", "service-ca.crt"
	}

	caName := *app.Spec.TLSCertificateAuthorityName

	// Case 3: System trust store - don't mount any CA
	if caName == "system-trust-store" {
		return "", ""
	}

	// Case 4: CA from environment bundle
	bundleSecretName := fmt.Sprintf("%s-ca-bundle", env.Name)
	fileName := fmt.Sprintf("%s.crt", caName)
	return bundleSecretName, fileName
}
