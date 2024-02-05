package featureflags

import (
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// AppSRE Unleash ver. 5.6.9
// https://github.com/app-sre/unleash/tree/64de4f47c57e84b9838f8f1f932822212caf55fb
var DefaultImageFeatureFlagsUnleash = "quay.io/app-sre/unleash:64de4f4"

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
