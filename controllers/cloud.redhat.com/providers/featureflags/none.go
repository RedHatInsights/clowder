package featureflags

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneFeatureFlagProvider struct {
	providers.Provider
}

// NewNoneFeatureFlagsProvider returns a new none feature flags provider object.
func NewNoneFeatureFlagsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneFeatureFlagProvider{Provider: *p}, nil
}

func (db *noneFeatureFlagProvider) EnvProvide() error {
	return nil
}

func (db *noneFeatureFlagProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
