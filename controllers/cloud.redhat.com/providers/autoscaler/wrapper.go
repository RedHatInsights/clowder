package autoscaler

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type autoScaleProvider struct {
	providers.Provider
	Config config.DatabaseConfig
}

// NewNoneDBProvider returns a new none db provider object.
func NewAutoScaleWrapper(p *providers.Provider) (providers.ClowderProvider, error) {
	return &autoScaleProvider{Provider: *p}, nil
}

func (db *autoScaleProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
