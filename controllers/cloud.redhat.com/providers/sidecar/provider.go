package sidecar

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

var DefaultImageSideCarTokenRefresher = "quay.io/observatorium/token-refresher:master-2023-09-20-f5e3403"                               // nolint:gosec
var DefaultImageSideCarOtelCollector = "ghcr.io/os-observability/redhat-opentelemetry-collector/redhat-opentelemetry-collector:0.107.0" // nolint:gosec

func GetTokenRefresherSidecar(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Sidecars.TokenRefresher.Image != "" {
		return env.Spec.Providers.Sidecars.TokenRefresher.Image
	}
	if clowderconfig.LoadedConfig.Images.TokenRefresher != "" {
		return clowderconfig.LoadedConfig.Images.TokenRefresher
	}
	return DefaultImageSideCarTokenRefresher
}

func GetOtelCollectorSidecar(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Sidecars.OtelCollector.Image != "" {
		return env.Spec.Providers.Sidecars.OtelCollector.Image
	}
	if clowderconfig.LoadedConfig.Images.OtelCollector != "" {
		return clowderconfig.LoadedConfig.Images.OtelCollector
	}
	return DefaultImageSideCarOtelCollector
}

func GetOtelCollectorConfigMap(env *crd.ClowdEnvironment, appName string) string {
	if env.Spec.Providers.Sidecars.OtelCollector.ConfigMap != "" {
		return env.Spec.Providers.Sidecars.OtelCollector.ConfigMap
	}
	return fmt.Sprintf("%s-otel-config", appName)
}

// ProvName sets the provider name identifier
var ProvName = "sidecar"

// GetEnd returns the correct end provider.
func GetSideCar(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewSidecarProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetSideCar, 98, ProvName)
}
