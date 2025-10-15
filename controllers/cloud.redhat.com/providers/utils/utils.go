// Package utils provides utility functions and helpers for Clowder providers
package utils // nolint:revive  // ignore meaningless name check

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

var defaultImageCaddySideCar = "quay.io/redhat-services-prod/hcm-eng-prod-tenant/crc-caddy-plugin:848bf12"
var defaultImageCaddyGateway = defaultImageCaddySideCar
var defaultImageMBOP = "quay.io/redhat-services-prod/hcc-fr-tenant/mbop/mbop:fbf8212"
var defaultImageMocktitlements = "quay.io/redhat-services-prod/hcm-eng-prod-tenant/mocktitlements-master/mocktitlements-master:f6b8612"
var defaultKeyCloakVersion = "23.0.1"
var defaultImageCaddyProxy = "quay.io/redhat-services-prod/hcm-eng-prod-tenant/caddy-ubi:84e5ba5"
var defaultImageKeyCloak = fmt.Sprintf("quay.io/keycloak/keycloak:%s", defaultKeyCloakVersion)
var defaultImageDatabasePG12 = "quay.io/cloudservices/postgresql-rds:12-2318dee"
var defaultImageDatabasePG13 = "quay.io/cloudservices/postgresql-rds:13-2318dee"
var defaultImageDatabasePG14 = "quay.io/cloudservices/postgresql-rds:14-2318dee"
var defaultImageDatabasePG15 = "quay.io/cloudservices/postgresql-rds:15-2318dee"
var defaultImageDatabasePG16 = "quay.io/cloudservices/postgresql-rds:16-759c25d"
var defaultImageInMemoryDB = "registry.redhat.io/rhel9/redis-6:1-199.1726663404"

// GetDefaultDatabaseImage returns the default image for the given PostgreSQL version
func GetDefaultDatabaseImage(version int32) (string, error) {
	var defaultImageDatabasePG = map[int32]string{
		16: defaultImageDatabasePG16,
		15: defaultImageDatabasePG15,
		14: defaultImageDatabasePG14,
		13: defaultImageDatabasePG13,
		12: defaultImageDatabasePG12,
	}

	image, ok := defaultImageDatabasePG[version]
	if !ok {
		return "", errors.NewClowderError(fmt.Sprintf("no default image for PostgreSQL version %d", version))
	}

	return image, nil
}

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

	dd.Spec.Template.Labels = labels

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

// GetInMemoryDBImage returns the in-memory database image for the environment
func GetInMemoryDBImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.InMemoryDB.Image != "" {
		return env.Spec.Providers.InMemoryDB.Image
	}
	if clowderconfig.LoadedConfig.Images.InMemoryDB != "" {
		return clowderconfig.LoadedConfig.Images.InMemoryDB
	}
	return defaultImageInMemoryDB
}

// GetCaddyGatewayImage returns the caddy gateway image to use in a given environment
func GetCaddyGatewayImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.CaddyGateway != "" {
		return env.Spec.Providers.Web.Images.CaddyGateway
	}
	if clowderconfig.LoadedConfig.Images.CaddyGateway != "" {
		return clowderconfig.LoadedConfig.Images.CaddyGateway
	}
	return defaultImageCaddyGateway
}

// GetCaddyImage returns the caddy image to use in a given environment
func GetCaddyImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.Caddy != "" {
		return env.Spec.Providers.Web.Images.Caddy
	}
	if clowderconfig.LoadedConfig.Images.Caddy != "" {
		return clowderconfig.LoadedConfig.Images.Caddy
	}
	return defaultImageCaddySideCar
}

// GetCaddyProxyImage returns the caddy image to use in a given environment
func GetCaddyProxyImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.CaddyProxy != "" {
		return env.Spec.Providers.Web.Images.CaddyProxy
	}
	if clowderconfig.LoadedConfig.Images.Caddy != "" {
		return clowderconfig.LoadedConfig.Images.CaddyProxy
	}
	return defaultImageCaddyProxy
}

// GetKeycloakImage returns the keycloak image to use in a given environment
func GetKeycloakImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.Keycloak != "" {
		return env.Spec.Providers.Web.Images.Keycloak
	}
	if clowderconfig.LoadedConfig.Images.Keycloak != "" {
		return clowderconfig.LoadedConfig.Images.Keycloak
	}
	return defaultImageKeyCloak
}

// GetMocktitlementsImage returns the mocktitlements image to use in a given environment
func GetMocktitlementsImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.Mocktitlements != "" {
		return env.Spec.Providers.Web.Images.Mocktitlements
	}
	if clowderconfig.LoadedConfig.Images.Mocktitlements != "" {
		return clowderconfig.LoadedConfig.Images.Mocktitlements
	}
	return defaultImageMocktitlements
}

// GetMockBOPImage returns the mock BOP image to use in a given environment
func GetMockBOPImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.Images.MockBOP != "" {
		return env.Spec.Providers.Web.Images.MockBOP
	}
	if clowderconfig.LoadedConfig.Images.MBOP != "" {
		return clowderconfig.LoadedConfig.Images.MBOP
	}
	return defaultImageMBOP
}

// GetKeycloakVersion returns the keycloak version to use in a given environment
func GetKeycloakVersion(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Web.KeycloakVersion != "" {
		return env.Spec.Providers.Web.KeycloakVersion
	}
	return defaultKeyCloakVersion
}

// GetClowderNamespace returns the namespace where Clowder is running
func GetClowderNamespace() (string, error) {
	clowderNsB, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")

	// CLOBBER the error here as this is our default
	if err != nil {
		return "clowder-system", nil
	}

	return string(clowderNsB), nil
}

// DebugLog logs a debug message with the provided logger and key-value pairs
func DebugLog(logger logr.Logger, msg string, keysAndValues ...interface{}) {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		logger.Info(msg, keysAndValues...)
	}
}

// KubeLinterAnnotations defines standard annotations to ignore specific kube-linter checks for Job pods
var KubeLinterAnnotations = map[string]string{
	"ignore-check.kube-linter.io/no-liveness-probe":  "probes not required on Job pods",
	"ignore-check.kube-linter.io/no-readiness-probe": "probes not required on Job pods",
}

// RCharSet defines the character set used for random string generation
const RCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// AddCertVolume adds a TLS certificate volume to the provided PodSpec
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

// DeploymentWithWebServices defines an interface for deployments that have web services configuration
type DeploymentWithWebServices interface {
	GetWebServices() crd.WebServices
}

// GetAPIPaths returns the API paths for a deployment with web services configuration
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

// SecretEnvVar represents an environment variable that references a secret key
type SecretEnvVar struct {
	Name string
	Key  string
}

// NewSecretEnvVar creates a new SecretEnvVar with the provided name and key
func NewSecretEnvVar(name, key string) SecretEnvVar {
	return SecretEnvVar{Name: name, Key: key}
}

// AppendEnvVarsFromSecret appends environment variables from a secret to the provided slice
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

// IsPublicTLSEnabled returns true if public TLS is enabled at the ClowdApp deployment level or at the ClowdEnvironment web provider level
func IsPublicTLSEnabled(deploymentWebConfig *crd.WebServices, envTLSConfig *crd.TLS) bool {
	if deploymentWebConfig.Public.TLS != nil {
		return *deploymentWebConfig.Public.TLS
	}
	return envTLSConfig.Enabled
}

// IsPrivateTLSEnabled returns true if private TLS is enabled at the ClowdApp deployment level or at the ClowdEnvironment web provider level
func IsPrivateTLSEnabled(deploymentWebConfig *crd.WebServices, envTLSConfig *crd.TLS) bool {
	if deploymentWebConfig.Private.TLS != nil {
		return *deploymentWebConfig.Private.TLS
	}
	return envTLSConfig.Enabled
}

// IsAnyTLSEnabled returns true if public OR private TLS is enabled at the ClowdApp deployment level or at the ClowdEnvironment web provider level
func IsAnyTLSEnabled(deploymentWebConfig *crd.WebServices, envTLSConfig *crd.TLS) bool {
	return IsPublicTLSEnabled(deploymentWebConfig, envTLSConfig) || IsPrivateTLSEnabled(deploymentWebConfig, envTLSConfig)
}
