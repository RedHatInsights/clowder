package autoscaler

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type simpleAutoScalerProvider struct {
	providers.Provider
	Config config.DatabaseConfig
}

// NewNoneDBProvider returns a new none db provider object.
func NewSimpleAutoScalerProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &simpleAutoScalerProvider{Provider: *p}, nil
}

func (db *simpleAutoScalerProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
