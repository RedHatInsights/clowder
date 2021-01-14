package featureflags

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type appInterfaceFeatureFlagProvider struct {
	p.Provider
	Config config.DatabaseConfig
}

func NewAppInterfaceFeatureFlagsProvider(p *p.Provider) (providers.ClowderProvider, error) {
	return &appInterfaceFeatureFlagProvider{Provider: *p}, nil
}

func (db *appInterfaceFeatureFlagProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
