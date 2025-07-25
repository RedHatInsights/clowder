package providers

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"

	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

var DefaultImageCaddySideCar = "quay.io/redhat-services-prod/hcm-eng-prod-tenant/crc-caddy-plugin:848bf12"
var DefaultImageCaddyGateway = DefaultImageCaddySideCar
var DefaultImageMBOP = "quay.io/cloudservices/mbop:4fbb291"
var DefaultImageMocktitlements = "quay.io/cloudservices/mocktitlements:745c249"
var DefaultKeyCloakVersion = "23.0.1"
var DefaultImageCaddyProxy = "quay.io/redhat-services-prod/hcm-eng-prod-tenant/caddy-ubi:5519eba"
var DefaultImageKeyCloak = fmt.Sprintf("quay.io/keycloak/keycloak:%s", DefaultKeyCloakVersion)
var DefaultImageDatabasePG12 = "quay.io/cloudservices/postgresql-rds:12-2318dee"
var DefaultImageDatabasePG13 = "quay.io/cloudservices/postgresql-rds:13-2318dee"
var DefaultImageDatabasePG14 = "quay.io/cloudservices/postgresql-rds:14-2318dee"
var DefaultImageDatabasePG15 = "quay.io/cloudservices/postgresql-rds:15-2318dee"
var DefaultImageDatabasePG16 = "quay.io/cloudservices/postgresql-rds:16-759c25d"
var DefaultImageInMemoryDB = "registry.redhat.io/rhel9/redis-6:1-199.1726663404"

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

	dd.Spec.Replicas = utils.Int32Ptr(1)
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
		{Name: "POSTGRESQL_ADMIN_PASSWORD", Value: cfg.AdminPassword},
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
		TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 5432},
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

func GetInMemoryDBImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.InMemoryDB.Image != "" {
		return env.Spec.Providers.InMemoryDB.Image
	}
	if clowderconfig.LoadedConfig.Images.InMemoryDB != "" {
		return clowderconfig.LoadedConfig.Images.InMemoryDB
	}
	return DefaultImageInMemoryDB
}

// GetCaddyImage returns the caddy image to use in a given environment
func GetCaddyGatewayImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.CaddyGateway != "" {
		return env.Spec.Providers.Web.Images.CaddyGateway
	}
	if clowderconfig.LoadedConfig.Images.CaddyGateway != "" {
		return clowderconfig.LoadedConfig.Images.CaddyGateway
	}
	return DefaultImageCaddyGateway
}

// GetCaddyImage returns the caddy image to use in a given environment
func GetCaddyImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.Caddy != "" {
		return env.Spec.Providers.Web.Images.Caddy
	}
	if clowderconfig.LoadedConfig.Images.Caddy != "" {
		return clowderconfig.LoadedConfig.Images.Caddy
	}
	return DefaultImageCaddySideCar
}

// GetCaddyProxyImage returns the caddy image to use in a given environment
func GetCaddyProxyImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.CaddyProxy != "" {
		return env.Spec.Providers.Web.Images.CaddyProxy
	}
	if clowderconfig.LoadedConfig.Images.Caddy != "" {
		return clowderconfig.LoadedConfig.Images.CaddyProxy
	}
	return DefaultImageCaddyProxy
}

// GetKeycloakImage returns the keycloak image to use in a given environment
func GetKeycloakImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.Keycloak != "" {
		return env.Spec.Providers.Web.Images.Keycloak
	}
	if clowderconfig.LoadedConfig.Images.Keycloak != "" {
		return clowderconfig.LoadedConfig.Images.Keycloak
	}
	return DefaultImageKeyCloak
}

// GetMocktitlementsImage returns the mocktitlements image to use in a given environment
func GetMocktitlementsImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.Mocktitlements != "" {
		return env.Spec.Providers.Web.Images.Mocktitlements
	}
	if clowderconfig.LoadedConfig.Images.Mocktitlements != "" {
		return clowderconfig.LoadedConfig.Images.Mocktitlements
	}
	return DefaultImageMocktitlements
}

// GetMockBOPImage returns the mock BOP image to use in a given environment
func GetMockBOPImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.MockBOP != "" {
		return env.Spec.Providers.Web.Images.MockBOP
	}
	if clowderconfig.LoadedConfig.Images.MBOP != "" {
		return clowderconfig.LoadedConfig.Images.MBOP
	}
	return DefaultImageMBOP
}

// GetKeycloakVersion returns the keycloak version to use in a given environment
func GetKeycloakVersion(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.KeycloakVersion != "" {
		return env.Spec.Providers.Web.KeycloakVersion
	}
	return DefaultKeyCloakVersion
}

func GetClowderNamespace() (string, error) {
	clowderNsB, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")

	// CLOBBER the error here as this is our default
	if err != nil {
		return "clowder-system", nil
	}

	return string(clowderNsB), nil
}

func DebugLog(logger logr.Logger, msg string, keysAndValues ...interface{}) {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		logger.Info(msg, keysAndValues...)
	}
}

var KubeLinterAnnotations = map[string]string{
	"ignore-check.kube-linter.io/no-liveness-probe":  "probes not required on Job pods",
	"ignore-check.kube-linter.io/no-readiness-probe": "probes not required on Job pods",
}

const RCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func AddCertVolume(d *core.PodSpec, dnn string) {
	d.Volumes = append(d.Volumes, core.Volume{
		Name: "tls-ca",
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: "openshift-service-ca.crt",
				},
			},
		},
	})
	for i, container := range d.Containers {
		vms := container.VolumeMounts
		if container.Name == dnn {
			vms = append(vms, core.VolumeMount{
				Name:      "tls-ca",
				ReadOnly:  true,
				MountPath: "/cdapp/certs",
			})
		}
		d.Containers[i].VolumeMounts = vms
	}

	for i, iContainer := range d.InitContainers {
		vms := iContainer.VolumeMounts
		vms = append(vms, core.VolumeMount{
			Name:      "tls-ca",
			ReadOnly:  true,
			MountPath: "/cdapp/certs",
		})
		d.InitContainers[i].VolumeMounts = vms
	}
}

type DeploymentWithWebServices interface {
	GetWebServices() v1alpha1.WebServices
}

func GetAPIPaths(deployment DeploymentWithWebServices, defaultPath string) []string {
	apiPaths := []string{}
	webServices := deployment.GetWebServices()
	if webServices.Public.APIPaths == nil {
		// singular apiPath is deprecated, use it only if apiPaths is undefined
		apiPath := webServices.Public.APIPath
		if apiPath == "" {
			apiPath = defaultPath
		}
		apiPaths = []string{fmt.Sprintf("/api/%s/", apiPath)}
	} else {
		// apiPaths was defined, use it and ignore 'apiPath'
		for _, path := range webServices.Public.APIPaths {
			// convert crd.APIPath array items into plain strings
			apiPaths = append(apiPaths, string(path))
		}
	}
	return apiPaths
}

type SecretEnvVar struct {
	Name string
	Key  string
}

func NewSecretEnvVar(name, key string) SecretEnvVar {
	return SecretEnvVar{Name: name, Key: key}
}

func AppendEnvVarsFromSecret(envvars []core.EnvVar, secName string, inputs ...SecretEnvVar) []core.EnvVar {
	for _, env := range inputs {
		newVar := core.EnvVar{
			Name: env.Name,
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: secName,
					},
					Key: env.Key,
				},
			},
		}
		envvars = append(envvars, newVar)
	}
	return envvars
}
