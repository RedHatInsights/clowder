package providers

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var DEFAULT_CADDY_IMAGE = "quay.io/cloudservices/crc-caddy-plugin:3ba6be7"
var DEFAULT_MBOP_IMAGE = "quay.io/cloudservices/mbop:0d3f99f"
var DEFAULT_MOCKTITLEMENTS_IMAGE = "quay.io/cloudservices/mocktitlements:130433d"
var DEFAULT_KEYCLOAK_VERSION = "15.0.2"
var DEFAULT_KEYCLOAK_IMAGE = fmt.Sprintf("quay.io/keycloak/keycloak:%s", DEFAULT_KEYCLOAK_VERSION)

// MakeLocalDB populates the given deployment object with the local DB struct.
func MakeLocalDB(dd *apps.Deployment, nn types.NamespacedName, baseResource obj.ClowdObject, extraLabels *map[string]string, cfg *config.DatabaseConfig, image string, usePVC bool, dbName string, res *core.ResourceRequirements) {
	labels := baseResource.GetLabels()
	labels["service"] = "db"

	for k, v := range *extraLabels {
		labels[k] = v
	}

	labler := utils.MakeLabeler(nn, labels, baseResource)
	labler(dd)

	var volSource core.VolumeSource
	if usePVC {
		volSource = core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: nn.Name,
			},
		}
		dd.Spec.Strategy.Type = apps.RecreateDeploymentStrategyType
		dd.Spec.Strategy.RollingUpdate = nil
	} else {
		volSource = core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		}
		dd.Spec.Strategy.Type = apps.RollingUpdateDeploymentStrategyType
		dd.Spec.Strategy.RollingUpdate = &apps.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "25%",
			},
			MaxSurge: &intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "25%",
			},
		}
	}

	dd.Spec.Replicas = common.Int32Ptr(1)
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{
		{
			Name:         nn.Name,
			VolumeSource: volSource,
		},
	}

	dd.Spec.Template.ObjectMeta.Labels = labels

	envVars := []core.EnvVar{
		{Name: "POSTGRESQL_USER", Value: cfg.Username},
		{Name: "POSTGRESQL_PASSWORD", Value: cfg.Password},
		{Name: "PGPASSWORD", Value: cfg.AdminPassword}, // Legacy for old db images can likely be removed soon
		{Name: "POSTGRESQL_MASTER_USER", Value: cfg.AdminUsername},
		{Name: "POSTGRESQL_MASTER_PASSWORD", Value: cfg.AdminPassword},
		{Name: "POSTGRESQL_DATABASE", Value: dbName},
	}
	ports := []core.ContainerPort{{
		Name:          "database",
		ContainerPort: 5432,
		Protocol:      core.ProtocolTCP,
	}}

	probeHandler := core.ProbeHandler{
		Exec: &core.ExecAction{
			Command: []string{
				"psql",
				"-U",
				"$(POSTGRESQL_USER)",
				"-d",
				"$(POSTGRESQL_DATABASE)",
				"-c",
				"SELECT 1",
			},
		},
	}

	livenessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 15,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 45,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	requestResource := sizing.GetDefaultResourceRequirements()

	if res != nil {
		requestResource = *res
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          image,
		Env:            envVars,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		Ports:          ports,
		Resources:      requestResource,
		VolumeMounts: []core.VolumeMount{{
			Name:      nn.Name,
			MountPath: "/var/lib/pgsql/data",
		}},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
}

// MakeLocalDBService populates the given service object with the local DB struct.
func MakeLocalDBService(s *core.Service, nn types.NamespacedName, baseResource obj.ClowdObject, extraLabels *map[string]string) {
	servicePorts := []core.ServicePort{{
		Name:       "database",
		Port:       5432,
		Protocol:   "TCP",
		TargetPort: intstr.FromInt(5432),
	}}
	labels := providers.Labels{"service": "db", "app": baseResource.GetClowdName()}
	for k, v := range *extraLabels {
		labels[k] = v
	}
	utils.MakeService(s, nn, labels, servicePorts, baseResource, false)
}

// MakeLocalDBPVC populates the given PVC object with the local DB struct.
func MakeLocalDBPVC(pvc *core.PersistentVolumeClaim, nn types.NamespacedName, baseResource obj.ClowdObject, capacity string) {
	utils.MakePVC(pvc, nn, providers.Labels{"service": "db", "app": baseResource.GetClowdName()}, capacity, baseResource)
}

// GetCaddyImage returns the caddy image to use in a given environment
func GetCaddyImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.CaddyImage != "" {
		return env.Spec.Providers.Web.CaddyImage
	}
	if clowderconfig.LoadedConfig.Images.Caddy != "" {
		return clowderconfig.LoadedConfig.Images.Caddy
	}
	return DEFAULT_CADDY_IMAGE
}

// GetKeycloakImage returns the keycloak image to use in a given environment
func GetKeycloakImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.KeycloakImage != "" {
		return env.Spec.Providers.Web.KeycloakImage
	}
	if clowderconfig.LoadedConfig.Images.Keycloak != "" {
		return clowderconfig.LoadedConfig.Images.Keycloak
	}
	return DEFAULT_KEYCLOAK_IMAGE
}

// GetMocktitlementsImage returns the mocktitlements image to use in a given environment
func GetMocktitlementsImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.MocktitlementsImage != "" {
		return env.Spec.Providers.Web.MocktitlementsImage
	}
	if clowderconfig.LoadedConfig.Images.Mocktitlements != "" {
		return clowderconfig.LoadedConfig.Images.Mocktitlements
	}
	return DEFAULT_MOCKTITLEMENTS_IMAGE
}

// GetMockBOPImage returns the mock BOP image to use in a given environment
func GetMockBOPImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.MockBOPImage != "" {
		return env.Spec.Providers.Web.MockBOPImage
	}
	if clowderconfig.LoadedConfig.Images.MBOP != "" {
		return clowderconfig.LoadedConfig.Images.MBOP
	}
	return DEFAULT_MBOP_IMAGE
}

// GetKeycloakVersion returns the keycloak version to use in a given environment
func GetKeycloakVersion(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.KeycloakVersion != "" {
		return env.Spec.Providers.Web.KeycloakVersion
	}
	// TODO: add config option in LoadedConfig for this?
	return DEFAULT_KEYCLOAK_VERSION
}
