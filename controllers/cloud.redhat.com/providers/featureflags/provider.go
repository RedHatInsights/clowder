package featureflags

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

var DefaultImageFeatureFlagsUnleash = "unleashorg/unleash-server:5.6.9"
var DefaultImageFeatureFlagsUnleashEdge = "unleashorg/unleash-edge:v19.6.3"

func GetFeatureFlagsUnleashImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.FeatureFlags.Images.Unleash != "" {
		return env.Spec.Providers.FeatureFlags.Images.Unleash
	}
	if clowderconfig.LoadedConfig.Images.FeatureFlagsUnleash != "" {
		return clowderconfig.LoadedConfig.Images.FeatureFlagsUnleash
	}
	return DefaultImageFeatureFlagsUnleash
}

func GetFeatureFlagsUnleashEdgeImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.FeatureFlags.Images.UnleashEdge != "" {
		return env.Spec.Providers.FeatureFlags.Images.UnleashEdge
	}
	if clowderconfig.LoadedConfig.Images.FeatureFlagsUnleashEdge != "" {
		return clowderconfig.LoadedConfig.Images.FeatureFlagsUnleashEdge
	}
	return DefaultImageFeatureFlagsUnleashEdge
}

// ProvName identifies the featureflags provider.
var ProvName = "featureflags"

// GetFeatureFlags returns the correct feature flags provider based on the environment.
func GetFeatureFlags(c *p.Provider) (p.ClowderProvider, error) {
	ffMode := c.Env.Spec.Providers.FeatureFlags.Mode
	switch ffMode {
	case "local":
		return NewLocalFeatureFlagsProvider(c)
	case "app-interface":
		return NewAppInterfaceFeatureFlagsProvider(c)
	case "none", "":
		return NewNoneFeatureFlagsProvider(c)
	default:
		errStr := fmt.Sprintf("No matching featureflags mode for %s", ffMode)
		return nil, errors.NewClowderError(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetFeatureFlags, 5, ProvName)
}
