package sidecar

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1/common"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	cronjobProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/cronjob"
	deployProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type sidecarProvider struct {
	providers.Provider
}

func NewSidecarProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &sidecarProvider{Provider: *p}, nil
}

func (sc *sidecarProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	for _, deployment := range app.Spec.Deployments {

		d := &apps.Deployment{}

		sc.Cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(&deployment))

		sideCarAdded := false

		for _, sidecar := range deployment.PodSpec.Sidecars {
			switch sidecar.Name {
			case "splunk":
				if sidecar.Enabled {
					sideCarAdded = true
					cont := getSplunk()
					if cont != nil {
						d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, *cont)
					}
				}
			case "token-refresher":
				if sidecar.Enabled {
					sideCarAdded = true
					cont := getTokenRefresher()
					if cont != nil {
						d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, *cont)
					}
				}
			default:
				return fmt.Errorf("%s is not a valid sidecar name", sidecar.Name)
			}

			if sideCarAdded {
				d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, core.Volume{
					Name: "splunk",
					VolumeSource: core.VolumeSource{
						Secret: &core.SecretVolumeSource{
							SecretName:  "splunk",
							DefaultMode: common.Int32Ptr(272),
						},
					},
				})
				sc.Cache.Update(deployProvider.CoreDeployment, d)
			}
		}
	}

	for _, cronJob := range app.Spec.Jobs {

		cj := &batch.CronJob{}

		sc.Cache.Get(cronjobProvider.CoreCronJob, cj, app.GetCronJobNamespacedName(&cronJob))

		sideCarAdded := false

		for _, sidecar := range cronJob.PodSpec.Sidecars {
			switch sidecar.Name {
			case "splunk":
				if sidecar.Enabled {
					sideCarAdded = true
					cont := getSplunk()
					if cont != nil {
						cj.Spec.JobTemplate.Spec.Template.Spec.Containers = append(cj.Spec.JobTemplate.Spec.Template.Spec.Containers, *cont)
					}
				}
			case "token-refresher":
				if sidecar.Enabled {
					sideCarAdded = true
					cont := getTokenRefresher()
					if cont != nil {
						cj.Spec.JobTemplate.Spec.Template.Spec.Containers = append(cj.Spec.JobTemplate.Spec.Template.Spec.Containers, *cont)
					}
				}
			default:
				return fmt.Errorf("%s is not a valid sidecar name", sidecar.Name)
			}

			if sideCarAdded {
				cj.Spec.JobTemplate.Spec.Template.Spec.Volumes = append(cj.Spec.JobTemplate.Spec.Template.Spec.Volumes, core.Volume{
					Name: "splunk",
					VolumeSource: core.VolumeSource{
						Secret: &core.SecretVolumeSource{
							SecretName:  "splunk",
							DefaultMode: common.Int32Ptr(272),
						},
					},
				})
				sc.Cache.Update(deployProvider.CoreDeployment, cj)
			}
		}
	}

	return nil
}

func getSplunk() *core.Container {
	cont := core.Container{}

	cont.Name = "splunk"
	cont.Image = "quay.io/cloudservices/rhsm-splunk-forwarder:8f72cfb"
	cont.VolumeMounts = []core.VolumeMount{{
		Name:      "splunk",
		MountPath: "/tls/splunk.pem",
		SubPath:   "splunk.pem",
	}}
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
	cont.Env = []core.EnvVar{{
		Name: "SPLUNKMETA_namespace",
		ValueFrom: &core.EnvVarSource{
			FieldRef: &core.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.namespace",
			},
		},
	}}

	return &cont
}

func getTokenRefresher() *core.Container {
	return nil
}
