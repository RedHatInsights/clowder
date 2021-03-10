package featureflags

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

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
		return nil, errors.New(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetFeatureFlags, 5, ProvName)
}
