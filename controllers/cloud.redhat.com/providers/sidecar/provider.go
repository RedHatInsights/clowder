package sidecar

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	core "k8s.io/api/core/v1"
)

// DefaultImageSideCarTokenRefresher defines the default token refresher sidecar image
var DefaultImageSideCarTokenRefresher = "quay.io/observatorium/token-refresher:master-2023-09-20-f5e3403" // nolint:gosec
// DefaultImageSideCarOtelCollector defines the default OpenTelemetry collector sidecar image
var DefaultImageSideCarOtelCollector = "ghcr.io/os-observability/redhat-opentelemetry-collector/redhat-opentelemetry-collector:0.107.0" // nolint:gosec

// GetTokenRefresherSidecar returns the token refresher sidecar image for the environment
func GetTokenRefresherSidecar(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Sidecars.TokenRefresher.Image != "" {
		return env.Spec.Providers.Sidecars.TokenRefresher.Image
	}
	if clowderconfig.LoadedConfig.Images.TokenRefresher != "" {
		return clowderconfig.LoadedConfig.Images.TokenRefresher
	}
	return DefaultImageSideCarTokenRefresher
}

// GetOtelCollectorSidecar returns the OpenTelemetry collector sidecar image for the environment
func GetOtelCollectorSidecar(env *crd.ClowdEnvironment, appSidecar *crd.Sidecar) string {
	// Priority: ClowdApp sidecar.image > ClowdEnvironment image > global config > default
	if appSidecar != nil && appSidecar.Image != "" {
		return appSidecar.Image
	}
	if env.Spec.Providers.Sidecars.OtelCollector.Image != "" {
		return env.Spec.Providers.Sidecars.OtelCollector.Image
	}
	if clowderconfig.LoadedConfig.Images.OtelCollector != "" {
		return clowderconfig.LoadedConfig.Images.OtelCollector
	}
	return DefaultImageSideCarOtelCollector
}

// GetOtelCollectorConfigMap returns the config map name for the OpenTelemetry collector
func GetOtelCollectorConfigMap(env *crd.ClowdEnvironment, appName string, appSidecar *crd.Sidecar) string {
	// Priority: ClowdApp sidecar.configMap > ClowdEnvironment configMap > default
	if appSidecar != nil && appSidecar.ConfigMap != "" {
		return appSidecar.ConfigMap
	}
	if env.Spec.Providers.Sidecars.OtelCollector.ConfigMap != "" {
		return env.Spec.Providers.Sidecars.OtelCollector.ConfigMap
	}
	return fmt.Sprintf("%s-otel-config", appName)
}

// ConvertEnvVars converts custom EnvVar type to Kubernetes EnvVar
func ConvertEnvVars(envVars []crd.EnvVar) []core.EnvVar {
	var coreEnvVars []core.EnvVar
	for _, envVar := range envVars {
		coreEnvVar := core.EnvVar{
			Name:  envVar.Name,
			Value: envVar.Value,
		}

		if envVar.ValueFrom != nil {
			coreEnvVar.ValueFrom = &core.EnvVarSource{}

			if envVar.ValueFrom.ConfigMapKeyRef != nil {
				coreEnvVar.ValueFrom.ConfigMapKeyRef = &core.ConfigMapKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: envVar.ValueFrom.ConfigMapKeyRef.Name,
					},
					Key:      envVar.ValueFrom.ConfigMapKeyRef.Key,
					Optional: envVar.ValueFrom.ConfigMapKeyRef.Optional,
				}
			}

			if envVar.ValueFrom.SecretKeyRef != nil {
				coreEnvVar.ValueFrom.SecretKeyRef = &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: envVar.ValueFrom.SecretKeyRef.Name,
					},
					Key:      envVar.ValueFrom.SecretKeyRef.Key,
					Optional: envVar.ValueFrom.SecretKeyRef.Optional,
				}
			}

			if envVar.ValueFrom.FieldRef != nil {
				coreEnvVar.ValueFrom.FieldRef = &core.ObjectFieldSelector{
					APIVersion: envVar.ValueFrom.FieldRef.APIVersion,
					FieldPath:  envVar.ValueFrom.FieldRef.FieldPath,
				}
				// Set default API version if not specified
				if coreEnvVar.ValueFrom.FieldRef.APIVersion == "" {
					coreEnvVar.ValueFrom.FieldRef.APIVersion = "v1"
				}
			}
		}

		coreEnvVars = append(coreEnvVars, coreEnvVar)
	}
	return coreEnvVars
}

// MergeEnvVars merges environment variables from environment and app level
// App-level env vars take precedence over environment-level env vars
// The order is preserved: environment-level variables first, then app-level variables
func MergeEnvVars(envVars []crd.EnvVar, appEnvVars []crd.EnvVar) []crd.EnvVar {
	// Create a map to track environment variable names for override detection
	envVarMap := make(map[string]bool)
	var result []crd.EnvVar

	// First add environment-level variables in order
	for _, envVar := range envVars {
		result = append(result, envVar)
		envVarMap[envVar.Name] = true
	}

	// Then add app-level variables in order, overriding existing ones
	for _, envVar := range appEnvVars {
		if envVarMap[envVar.Name] {
			// Override existing environment variable by finding and replacing it
			for i, existing := range result {
				if existing.Name == envVar.Name {
					result[i] = envVar
					break
				}
			}
		} else {
			// Add new app-level variable
			result = append(result, envVar)
			envVarMap[envVar.Name] = true
		}
	}

	return result
}

// ProvName sets the provider name identifier
var ProvName = "sidecar"

// GetSideCar returns the correct sidecar provider.
func GetSideCar(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewSidecarProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetSideCar, 98, ProvName)
}
