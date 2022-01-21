package deployment

import (
	"fmt"
	"strconv"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (dp *deploymentProvider) makeDeployment(deployment crd.Deployment, app *crd.ClowdApp) error {

	d := &apps.Deployment{}
	nn := app.GetDeploymentNamespacedName(&deployment)

	if err := dp.Cache.Create(CoreDeployment, nn, d); err != nil {
		return err
	}

	if err := initDeployment(app, dp.Env, d, nn, deployment); err != nil {
		return err
	}

	if err := dp.Cache.Update(CoreDeployment, d); err != nil {
		return err
	}

	return nil
}

func initDeployment(app *crd.ClowdApp, env *crd.ClowdEnvironment, d *apps.Deployment, nn types.NamespacedName, deployment crd.Deployment) error {
	labels := app.GetLabels()
	labels["pod"] = nn.Name
	app.SetObjectMeta(d, crd.Name(nn.Name), crd.Labels(labels))

	d.Kind = "Deployment"

	pod := deployment.PodSpec
	d.Spec.Template.SetAnnotations(make(map[string]string))
	if env.Spec.Providers.Web.Mode == "local" && (deployment.WebServices.Public.Enabled || bool(deployment.Web)) {
		d.Spec.Template.Annotations["clowder/authsidecar-image"] = "a76bb81"
		d.Spec.Template.Annotations["clowder/authsidecar-enabled"] = "true"
		d.Spec.Template.Annotations["clowder/authsidecar-port"] = strconv.Itoa(int(env.Spec.Providers.Web.Port))
		d.Spec.Template.Annotations["clowder/authsidecar-config"] = fmt.Sprintf("caddy-config-%s-%s", app.Name, deployment.Name)
	}

	if deployment.AutoScaler != nil {
		// let autoscaler scale without reconciliation re-writing the replicas
		if d.Spec.Replicas == nil {
			// Replicas is nil during deployment initialisation
			d.Spec.Replicas = deployment.MinReplicas
		} else if deployment.MinReplicas != nil && *d.Spec.Replicas < *deployment.MinReplicas {
			// Reset replicas to minReplicas if it somehow falls below minReplicas
			d.Spec.Replicas = deployment.MinReplicas
		}
	} else {
		d.Spec.Replicas = deployment.MinReplicas
	}

	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	d.Spec.Template.ObjectMeta.Labels = labels
	d.Spec.Strategy = apps.DeploymentStrategy{
		Type: apps.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &apps.RollingUpdateDeployment{
			MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: string("25%")},
			MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: string("25%")},
		},
	}
	d.Spec.ProgressDeadlineSeconds = common.Int32Ptr(600)

	if !deployment.WebServices.Public.Enabled {
		if deployment.DeploymentStrategy != nil && deployment.DeploymentStrategy.PrivateStrategy != "" {
			d.Spec.Strategy = apps.DeploymentStrategy{
				Type: deployment.DeploymentStrategy.PrivateStrategy,
			}
		} else {
			d.Spec.Strategy = apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			}
		}
	}

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

	baseProbe := core.Probe{
		Handler: core.Handler{
			HTTPGet: &core.HTTPGetAction{
				Path:   "/healthz",
				Scheme: "HTTP",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: env.Spec.Providers.Web.Port,
				},
			},
		},
		FailureThreshold:    3,
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		TimeoutSeconds:      1,
	}
	if pod.LivenessProbe != nil {
		livenessProbe = *pod.LivenessProbe

		if livenessProbe.SuccessThreshold == 0 {
			livenessProbe.SuccessThreshold = 1
		}
		if livenessProbe.TimeoutSeconds == 0 {
			livenessProbe.TimeoutSeconds = 1
		}
		if livenessProbe.PeriodSeconds == 0 {
			livenessProbe.PeriodSeconds = 10
		}
		if livenessProbe.FailureThreshold == 0 {
			livenessProbe.FailureThreshold = 3
		}
	} else if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
		livenessProbe = baseProbe
	}
	if pod.ReadinessProbe != nil {
		readinessProbe = *pod.ReadinessProbe

		if readinessProbe.SuccessThreshold == 0 {
			readinessProbe.SuccessThreshold = 1
		}
		if readinessProbe.TimeoutSeconds == 0 {
			readinessProbe.TimeoutSeconds = 1
		}
		if readinessProbe.PeriodSeconds == 0 {
			readinessProbe.PeriodSeconds = 10
		}
		if readinessProbe.FailureThreshold == 0 {
			readinessProbe.FailureThreshold = 3
		}
	} else if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
		readinessProbe = baseProbe
		readinessProbe.InitialDelaySeconds = 45
	}

	c := core.Container{
		Name:                     nn.Name,
		Image:                    pod.Image,
		Command:                  pod.Command,
		Args:                     pod.Args,
		Env:                      envvar,
		Resources:                ProcessResources(&pod, env),
		VolumeMounts:             pod.VolumeMounts,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
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

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "config-secret",
		MountPath: "/cdapp/",
	})

	d.Spec.Template.Spec.Containers = []core.Container{c}

	ics, err := ProcessInitContainers(nn, &c, pod.InitContainers)

	if err != nil {
		return err
	}

	d.Spec.Template.Spec.InitContainers = ics

	d.Spec.Template.Spec.Volumes = pod.Volumes
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				DefaultMode: common.Int32Ptr(420),
				SecretName:  app.ObjectMeta.Name,
			},
		},
	})

	for _, vol := range d.Spec.Template.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			d.Spec.Strategy = apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			}
			break
		} else if vol.VolumeSource.ConfigMap != nil && (vol.VolumeSource.ConfigMap.DefaultMode == nil || *vol.VolumeSource.ConfigMap.DefaultMode == 0) {
			vol.VolumeSource.ConfigMap.DefaultMode = common.Int32Ptr(420)
		} else if vol.VolumeSource.Secret != nil && (vol.VolumeSource.Secret.DefaultMode == nil || *vol.VolumeSource.Secret.DefaultMode == 0) {
			vol.VolumeSource.Secret.DefaultMode = common.Int32Ptr(420)
		}
	}

	ApplyPodAntiAffinity(&d.Spec.Template)

	return nil
}

// ProcessInitContainers returns a container object which has been processed from the container spec.
func ProcessInitContainers(nn types.NamespacedName, c *core.Container, ics []crd.InitContainer) ([]core.Container, error) {
	if len(ics) == 0 {
		return []core.Container{}, nil
	}
	containerList := make([]core.Container, len(ics))

	for i, ic := range ics {

		image := c.Image
		if ic.Image != "" {
			image = ic.Image
		}

		if len(ics) > 1 && ic.Name == "" {
			return []core.Container{}, fmt.Errorf("multiple init containers must have name")
		}

		name := nn.Name
		if ic.Name != "" {
			name = ic.Name
		}

		icStruct := core.Container{
			Name:                     name + "-init",
			Image:                    image,
			Command:                  ic.Command,
			Args:                     ic.Args,
			Resources:                c.Resources,
			VolumeMounts:             c.VolumeMounts,
			ImagePullPolicy:          c.ImagePullPolicy,
			TerminationMessagePath:   "/dev/termination-log",
			TerminationMessagePolicy: core.TerminationMessageReadFile,
		}

		if ic.InheritEnv {
			// The idea here is that you can override the inherited values by
			// setting them on the initContainer env
			icStruct.Env = []core.EnvVar{}
			usedVars := make(map[string]bool)

			for _, iEnvVar := range ic.Env {
				icStruct.Env = append(icStruct.Env, core.EnvVar{
					Name:      iEnvVar.Name,
					Value:     iEnvVar.Value,
					ValueFrom: iEnvVar.ValueFrom,
				})
				usedVars[iEnvVar.Name] = true
			}
			for _, e := range c.Env {
				if _, ok := usedVars[e.Name]; !ok {
					icStruct.Env = append(icStruct.Env, core.EnvVar{
						Name:      e.Name,
						Value:     e.Value,
						ValueFrom: e.ValueFrom,
					})
				}
			}
		} else {

			icStruct.Env = append(
				icStruct.Env, core.EnvVar{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"},
			)

			for _, envvar := range ic.Env {
				icStruct.Env = append(icStruct.Env, core.EnvVar{
					Name:      envvar.Name,
					Value:     envvar.Value,
					ValueFrom: envvar.ValueFrom,
				})
			}
		}

		containerList[i] = icStruct
	}
	return containerList, nil
}

// ApplyPodAntiAffinity applies pod anti affinity rules to a pod template
func ApplyPodAntiAffinity(t *core.PodTemplateSpec) {
	labelSelector := &metav1.LabelSelector{MatchLabels: t.Labels}
	t.Spec.Affinity = &core.Affinity{PodAntiAffinity: &core.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []core.WeightedPodAffinityTerm{{
			Weight: 100,
			PodAffinityTerm: core.PodAffinityTerm{
				LabelSelector: labelSelector,
				TopologyKey:   "failure-domain.beta.kubernetes.io/zone",
			},
		}, {
			Weight: 99,
			PodAffinityTerm: core.PodAffinityTerm{
				LabelSelector: labelSelector,
				TopologyKey:   "kubernetes.io/hostname",
			},
		}},
	}}
}

// ProcessResources takes a pod spec and a clowd environment and returns the resource requirements
// object.
func ProcessResources(pod *crd.PodSpec, env *crd.ClowdEnvironment) core.ResourceRequirements {
	var lcpu, lmemory, rcpu, rmemory resource.Quantity
	nullCPU := resource.Quantity{Format: resource.DecimalSI}
	nullMemory := resource.Quantity{Format: resource.BinarySI}

	if *pod.Resources.Limits.Cpu() != nullCPU {
		lcpu = pod.Resources.Limits["cpu"]
	} else {
		lcpu = env.Spec.ResourceDefaults.Limits["cpu"]
	}

	if *pod.Resources.Limits.Memory() != nullMemory {
		lmemory = pod.Resources.Limits["memory"]
	} else {
		lmemory = env.Spec.ResourceDefaults.Limits["memory"]
	}

	if *pod.Resources.Requests.Cpu() != nullCPU {
		rcpu = pod.Resources.Requests["cpu"]
	} else {
		rcpu = env.Spec.ResourceDefaults.Requests["cpu"]
	}

	if *pod.Resources.Requests.Memory() != nullMemory {
		rmemory = pod.Resources.Requests["memory"]
	} else {
		rmemory = env.Spec.ResourceDefaults.Requests["memory"]
	}

	return core.ResourceRequirements{
		Limits: core.ResourceList{
			"cpu":    lcpu,
			"memory": lmemory,
		},
		Requests: core.ResourceList{
			"cpu":    rcpu,
			"memory": rmemory,
		},
	}
}
