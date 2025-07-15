package deployment

import (
	"fmt"
	"strconv"
	"strings"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
)

const (
	TerminationLogPath = "/dev/termination-log"
)

func (dp *deploymentProvider) makeDeployment(deployment *crd.Deployment, app *crd.ClowdApp) error {

	d := &apps.Deployment{}
	nn := app.GetDeploymentNamespacedName(deployment)

	if err := dp.Cache.Create(CoreDeployment, nn, d); err != nil {
		return err
	}

	if err := initDeployment(app, dp.Env, d, nn, deployment); err != nil {
		return err
	}

	return dp.Cache.Update(CoreDeployment, d)
}

func setLocalAnnotations(env *crd.ClowdEnvironment, deployment *crd.Deployment, d *apps.Deployment, app *crd.ClowdApp) {
	if env.Spec.Providers.Web.Mode == "local" && (deployment.WebServices.Public.Enabled || bool(deployment.Web)) {
		annotations := map[string]string{
			"clowder/authsidecar-image":   provutils.GetCaddyImage(env),
			"clowder/authsidecar-enabled": "true",
			"clowder/authsidecar-port":    strconv.Itoa(int(env.Spec.Providers.Web.Port)),
			"clowder/authsidecar-config":  fmt.Sprintf("caddy-config-%s-%s", app.Name, deployment.Name),
		}
		utils.UpdateAnnotations(&d.Spec.Template, annotations)
	}

}

func setMinReplicas(deployment *crd.Deployment, d *apps.Deployment) {
	replicaCount := deployment.GetReplicaCount()
	// If deployment doesn't have minReplicas set, bail
	if replicaCount == nil {
		return
	}

	// Handle the special case of minReplicas being set to 0 used for manual scale down
	if *replicaCount == 0 {
		d.Spec.Replicas = utils.Int32Ptr(0)
		return
	}

	// No sense in running all these conditionals if desired state and observed state match
	if d.Spec.Replicas != nil && (*d.Spec.Replicas >= *replicaCount) {
		// if deployment has an autoscaler, just keep the replica count the same it currently is
		// since it has at least the desired count
		if deployment.HasAutoScaler() {
			return
		}
		// reset the replica count when there is no autoscaler. this will scale down a deployment that
		// has more than desired count
		d.Spec.Replicas = replicaCount
	}

	// If the spec has nil replicas or the spec replicas are less than the deployment replicas
	// then set the spec replicas to the deployment replicas
	if d.Spec.Replicas == nil || (*d.Spec.Replicas < *replicaCount) {
		// Reset replicas to minReplicas if it somehow falls below minReplicas
		d.Spec.Replicas = replicaCount
	}

}

func setDeploymentStrategy(deployment *crd.Deployment, d *apps.Deployment) {
	if !deployment.WebServices.Public.Enabled {
		if deployment.DeploymentStrategy != nil && deployment.DeploymentStrategy.PrivateStrategy != "" {
			d.Spec.Strategy = apps.DeploymentStrategy{
				Type: deployment.DeploymentStrategy.PrivateStrategy,
			}
		} else {
			d.Spec.Strategy = apps.DeploymentStrategy{
				Type: apps.RollingUpdateDeploymentStrategyType,
			}
		}
	}
}

func makeBaseProbe(env *crd.ClowdEnvironment) core.Probe {
	return core.Probe{
		ProbeHandler: core.ProbeHandler{
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
}

func setLivenessProbe(pod *crd.PodSpec, deployment *crd.Deployment, env *crd.ClowdEnvironment, c *core.Container) {
	livenessProbe := core.Probe{}

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
		c.LivenessProbe = &livenessProbe
	} else if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
		livenessProbe = makeBaseProbe(env)
		c.LivenessProbe = &livenessProbe
	}
}

func setReadinessProbe(pod *crd.PodSpec, deployment *crd.Deployment, env *crd.ClowdEnvironment, c *core.Container) {
	readinessProbe := core.Probe{}
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

		c.ReadinessProbe = &readinessProbe
	} else if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
		readinessProbe := makeBaseProbe(env)
		readinessProbe.InitialDelaySeconds = 45
		c.ReadinessProbe = &readinessProbe
	}
}

func setImagePullPolicy(env *crd.ClowdEnvironment, c *core.Container) {
	if !env.Spec.Providers.Deployment.OmitPullPolicy {
		c.ImagePullPolicy = core.PullIfNotPresent
		return
	}
	imageComponents := strings.Split(c.Image, ":")
	if len(imageComponents) > 1 {
		if imageComponents[1] == "latest" {
			c.ImagePullPolicy = core.PullAlways
		} else {
			c.ImagePullPolicy = core.PullIfNotPresent
		}
		return
	}
	c.ImagePullPolicy = core.PullIfNotPresent

}

func loadEnvVars(pod *crd.PodSpec) []core.EnvVar {
	envvars := pod.Env
	envvars = append(envvars, core.EnvVar{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"})

	for _, envvar := range envvars {
		if envvar.ValueFrom != nil {
			if envvar.ValueFrom.FieldRef != nil {
				if envvar.ValueFrom.FieldRef.APIVersion == "" {
					envvar.ValueFrom.FieldRef.APIVersion = "v1"
				}
			}
		}
	}

	return envvars
}

func initDeployment(app *crd.ClowdApp, env *crd.ClowdEnvironment, d *apps.Deployment, nn types.NamespacedName, deployment *crd.Deployment) error {
	labels := app.GetLabels()
	labels["pod"] = nn.Name
	app.SetObjectMeta(d, crd.Name(nn.Name), crd.Labels(labels))

	d.Kind = "Deployment"

	pod := deployment.PodSpec

	utils.UpdateAnnotations(d, app.ObjectMeta.Annotations, deployment.Metadata.Annotations)

	setLocalAnnotations(env, deployment, d, app)

	setMinReplicas(deployment, d)

	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	d.Spec.Template.ObjectMeta.Labels = labels
	d.Spec.Strategy = apps.DeploymentStrategy{
		Type: apps.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &apps.RollingUpdateDeployment{
			MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: string("25%")},
			MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: string("25%")},
		},
	}
	d.Spec.ProgressDeadlineSeconds = utils.Int32Ptr(600)

	utils.UpdateAnnotations(&d.Spec.Template, pod.Metadata.Annotations)

	setDeploymentStrategy(deployment, d)

	c := core.Container{
		Name:                     nn.Name,
		Image:                    pod.Image,
		Command:                  pod.Command,
		Args:                     pod.Args,
		Env:                      loadEnvVars(&pod),
		Resources:                ProcessResources(&pod, env),
		VolumeMounts:             pod.VolumeMounts,
		TerminationMessagePath:   TerminationLogPath,
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
		Lifecycle:                pod.Lifecycle,
	}

	setLivenessProbe(&pod, deployment, env, &c)
	setReadinessProbe(&pod, deployment, env, &c)
	setImagePullPolicy(env, &c)

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "config-secret",
		MountPath: "/cdapp/",
	})

	d.Spec.Template.Spec.Containers = []core.Container{c}

	ics, err := ProcessInitContainers(nn, &c, pod.InitContainers)

	if err != nil {
		return err
	}

	if pod.MachinePool != "" {
		d.Spec.Template.Spec.Tolerations = []core.Toleration{{
			Key:      pod.MachinePool,
			Effect:   core.TaintEffectNoSchedule,
			Operator: core.TolerationOpEqual,
			Value:    "true",
		}}
	} else {
		d.Spec.Template.Spec.Tolerations = []core.Toleration{}
	}

	d.Spec.Template.Spec.InitContainers = ics

	d.Spec.Template.Spec.TerminationGracePeriodSeconds = pod.TerminationGracePeriodSeconds

	d.Spec.Template.Spec.Volumes = pod.Volumes
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-secret",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				DefaultMode: utils.Int32Ptr(420),
				SecretName:  app.ObjectMeta.Name,
			},
		},
	})

	for _, vol := range d.Spec.Template.Spec.Volumes {
		v := vol
		setRecreateDeploymentStrategyForPVCs(vol, d)
		setVolumeSourceConfigMapDefaultMode(&v)
		setVolumeSourceSecretDefaultMode(&v)
	}

	ApplyPodAntiAffinity(&d.Spec.Template)

	return nil
}

func setRecreateDeploymentStrategyForPVCs(vol core.Volume, d *apps.Deployment) {
	if vol.VolumeSource.PersistentVolumeClaim == nil {
		return
	}
	d.Spec.Strategy = apps.DeploymentStrategy{
		Type: apps.RecreateDeploymentStrategyType,
	}
}

func setVolumeSourceConfigMapDefaultMode(vol *core.Volume) {
	if vol.VolumeSource.PersistentVolumeClaim != nil {
		return
	}
	if vol.VolumeSource.ConfigMap != nil && vol.VolumeSource.Secret == nil && (vol.VolumeSource.ConfigMap.DefaultMode == nil || *vol.VolumeSource.ConfigMap.DefaultMode == 0) {
		vol.VolumeSource.ConfigMap.DefaultMode = utils.Int32Ptr(420)
	}
}

func setVolumeSourceSecretDefaultMode(vol *core.Volume) {
	if vol.VolumeSource.PersistentVolumeClaim != nil {
		return
	}
	if vol.VolumeSource.ConfigMap == nil && vol.VolumeSource.Secret != nil && (vol.VolumeSource.Secret.DefaultMode == nil || *vol.VolumeSource.Secret.DefaultMode == 0) {
		vol.VolumeSource.Secret.DefaultMode = utils.Int32Ptr(420)
	}
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
			TerminationMessagePath:   TerminationLogPath,
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
