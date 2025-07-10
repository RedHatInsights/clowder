package sidecar

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	cronjobProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/cronjob"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type sidecarProvider struct {
	providers.Provider
}

func NewSidecarProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &sidecarProvider{Provider: *p}, nil
}

func (sc *sidecarProvider) EnvProvide() error {
	return nil
}

func (sc *sidecarProvider) Provide(app *crd.ClowdApp) error {
	for _, deployment := range app.Spec.Deployments {
		innerDeployment := deployment
		d := &apps.Deployment{}

		if err := sc.Cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(&innerDeployment)); err != nil {
			return err
		}

		for _, sidecar := range innerDeployment.PodSpec.Sidecars {
			switch sidecar.Name {
			case "token-refresher":
				if sidecar.Enabled && sc.Env.Spec.Providers.Sidecars.TokenRefresher.Enabled {
					cont := getTokenRefresher(app.Name)
					if cont != nil {
						d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, *cont)
					}
				}
			case "otel-collector":
				if sidecar.Enabled && sc.Env.Spec.Providers.Sidecars.OtelCollector.Enabled {
					cont := getOtelCollector(app.Name, sc.Env)
					if cont != nil {
						d.Spec.Template.Spec.InitContainers = append(d.Spec.Template.Spec.InitContainers, *cont)
						d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, core.Volume{
							Name: fmt.Sprintf("%s-otel-config", app.Name),
							VolumeSource: core.VolumeSource{
								ConfigMap: &core.ConfigMapVolumeSource{
									LocalObjectReference: core.LocalObjectReference{
										Name: fmt.Sprintf("%s-otel-config", app.Name),
									},
									Optional: utils.TruePtr(),
								},
							},
						})
					}
				}
			default:
				return fmt.Errorf("%s is not a valid sidecar name", sidecar.Name)
			}
		}

		if err := sc.Cache.Update(deployProvider.CoreDeployment, d); err != nil {
			return err
		}
	}

	for _, cronJob := range app.Spec.Jobs {
		innerCronJob := cronJob
		if innerCronJob.Schedule == "" || innerCronJob.Disabled {
			continue
		}
		cj := &batch.CronJob{}

		if err := sc.Cache.Get(cronjobProvider.CoreCronJob, cj, app.GetCronJobNamespacedName(&innerCronJob)); err != nil {
			return err
		}

		for _, sidecar := range innerCronJob.PodSpec.Sidecars {
			switch sidecar.Name {
			case "token-refresher":
				if sidecar.Enabled && sc.Env.Spec.Providers.Sidecars.TokenRefresher.Enabled {
					cont := getTokenRefresher(app.Name)
					if cont != nil {
						cj.Spec.JobTemplate.Spec.Template.Spec.Containers = append(cj.Spec.JobTemplate.Spec.Template.Spec.Containers, *cont)
					}
				}
			case "otel-collector":
				if sidecar.Enabled && sc.Env.Spec.Providers.Sidecars.OtelCollector.Enabled {
					cont := getOtelCollector(app.Name, sc.Env)
					if cont != nil {
						cj.Spec.JobTemplate.Spec.Template.Spec.InitContainers = append(cj.Spec.JobTemplate.Spec.Template.Spec.InitContainers, *cont)
						cj.Spec.JobTemplate.Spec.Template.Spec.Volumes = append(cj.Spec.JobTemplate.Spec.Template.Spec.Volumes, core.Volume{
							Name: fmt.Sprintf("%s-otel-config", app.Name),
							VolumeSource: core.VolumeSource{
								ConfigMap: &core.ConfigMapVolumeSource{
									LocalObjectReference: core.LocalObjectReference{
										Name: fmt.Sprintf("%s-otel-config", app.Name),
									},
									Optional: utils.TruePtr(),
								},
							},
						})
					}
				}
			default:
				return fmt.Errorf("%s is not a valid sidecar name", sidecar.Name)
			}
		}

		if err := sc.Cache.Update(cronjobProvider.CoreCronJob, cj); err != nil {
			return err
		}
	}

	return nil
}

func getTokenRefresher(appName string) *core.Container {
	cont := core.Container{}

	cont.Name = "token-refresher"
	cont.Image = DefaultImageSideCarTokenRefresher
	cont.Args = []string{
		"--oidc.audience=observatorium-telemeter",
		"--oidc.client-id=$(CLIENT_ID)",
		"--oidc.client-secret=$(CLIENT_SECRET)",
		"--oidc.issuer-url=$(ISSUER_URL)",
		"--url=$(URL)",
		"--web.listen=:8082",
		"--scope=$(SCOPE)",
	}
	cont.TerminationMessagePath = "/dev/termination-log"
	cont.TerminationMessagePolicy = core.TerminationMessageReadFile
	cont.ImagePullPolicy = core.PullIfNotPresent
	cont.Resources = core.ResourceRequirements{
		Limits: core.ResourceList{
			"cpu":    resource.MustParse("100m"),
			"memory": resource.MustParse("150Mi"),
		},
		Requests: core.ResourceList{
			"cpu":    resource.MustParse("50m"),
			"memory": resource.MustParse("128Mi"),
		},
	}

	envVars := []core.EnvVar{}

	envVars = provutils.AppendEnvVarsFromSecret(envVars, fmt.Sprintf("%s-token-refresher", appName),
		provutils.NewSecretEnvVar("CLIENT_ID", "CLIENT_ID"),
		provutils.NewSecretEnvVar("CLIENT_SECRET", "CLIENT_SECRET"),
		provutils.NewSecretEnvVar("ISSUER_URL", "ISSUER_URL"),
		provutils.NewSecretEnvVar("SCOPE", "SCOPE"),
		provutils.NewSecretEnvVar("URL", "URL"),
	)

	cont.Env = envVars

	return &cont
}

func getOtelCollector(appName string, env *crd.ClowdEnvironment) *core.Container {
	port := int32(13133)
	probeHandler := core.ProbeHandler{
		HTTPGet: &core.HTTPGetAction{
			Path: "/",
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port,
			},
		},
	}

	livenessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      4,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 40,
		TimeoutSeconds:      4,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	cont := core.Container{}

	restartPolicy := core.ContainerRestartPolicyAlways
	cont.Name = "otel-collector"
	cont.Image = GetOtelCollectorSidecar(env)
	cont.Args = []string{}
	cont.TerminationMessagePath = "/dev/termination-log"
	cont.TerminationMessagePolicy = core.TerminationMessageReadFile
	cont.ImagePullPolicy = core.PullIfNotPresent
	cont.RestartPolicy = &restartPolicy
	cont.Resources = core.ResourceRequirements{
		Limits: core.ResourceList{
			"cpu":    resource.MustParse("500m"),
			"memory": resource.MustParse("1024Mi"),
		},
		Requests: core.ResourceList{
			"cpu":    resource.MustParse("250m"),
			"memory": resource.MustParse("512Mi"),
		},
	}
	cont.VolumeMounts = []core.VolumeMount{{
		Name:      fmt.Sprintf("%s-otel-config", appName),
		MountPath: "/etc/otelcol/",
	}}
	cont.LivenessProbe = &livenessProbe
	cont.ReadinessProbe = &readinessProbe

	return &cont
}
