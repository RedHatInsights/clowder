package featureflags

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneFeatureFlagProvider struct {
	providers.Provider
}

// NewNoneFeatureFlagsProvider returns a new none feature flags provider object.
func NewNoneFeatureFlagsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneFeatureFlagProvider{Provider: *p}, nil
}

func (db *noneFeatureFlagProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
