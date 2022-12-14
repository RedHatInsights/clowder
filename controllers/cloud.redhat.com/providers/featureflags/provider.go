package featureflags

import (
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

var DefaultImageFeatureFlagsUnleash = "quay.io/cloudservices/unleash-docker:3.9"

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
