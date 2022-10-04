package sidecar

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	cronjobProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/cronjob"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

		d := &apps.Deployment{}

		if err := sc.Cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(&deployment)); err != nil {
			return err
		}

		for _, sidecar := range deployment.PodSpec.Sidecars {
			switch sidecar.Name {
			case "token-refresher":
				if sidecar.Enabled && sc.Env.Spec.Providers.Sidecars.TokenRefresher.Enabled {
					cont := getTokenRefresher(app.Name)
					if cont != nil {
						d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, *cont)
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
		if cronJob.Schedule == "" {
			continue
		}
		cj := &batch.CronJob{}

		if err := sc.Cache.Get(cronjobProvider.CoreCronJob, cj, app.GetCronJobNamespacedName(&cronJob)); err != nil {
			return err
		}

		for _, sidecar := range cronJob.PodSpec.Sidecars {
			switch sidecar.Name {
			case "token-refresher":
				if sidecar.Enabled && sc.Env.Spec.Providers.Sidecars.TokenRefresher.Enabled {
					cont := getTokenRefresher(app.Name)
					if cont != nil {
						cj.Spec.JobTemplate.Spec.Template.Spec.Containers = append(cj.Spec.JobTemplate.Spec.Template.Spec.Containers, *cont)
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
	cont.Image = IMAGE_SIDECAR_TOKEN_REFRESHER
	cont.Args = []string{
		"--oidc.audience=observatorium-telemeter",
		"--oidc.client-id=$(CLIENT_ID)",
		"--oidc.client-secret=$(CLIENT_SECRET)",
		"--oidc.issuer-url=$(ISSUER_URL)",
		"--url=$(URL)",
		"--web.listen=:8082",
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
	cont.Env = []core.EnvVar{
		{
			Name: "CLIENT_ID",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: fmt.Sprintf("%s-token-refresher", appName),
					},
					Key: "CLIENT_ID",
				},
			},
		},
		{
			Name: "CLIENT_SECRET",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: fmt.Sprintf("%s-token-refresher", appName),
					},
					Key: "CLIENT_SECRET",
				},
			},
		},
		{
			Name: "ISSUER_URL",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: fmt.Sprintf("%s-token-refresher", appName),
					},
					Key: "ISSUER_URL",
				},
			},
		},
		{
			Name: "URL",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: fmt.Sprintf("%s-token-refresher", appName),
					},
					Key: "URL",
				},
			},
		},
	}

	return &cont
}
