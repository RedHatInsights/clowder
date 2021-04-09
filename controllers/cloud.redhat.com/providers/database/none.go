package database

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneDbProvider struct {
	providers.Provider
	Config config.DatabaseConfig
}

// NewNoneDBProvider returns a new none db provider object.
func NewNoneDBProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneDbProvider{Provider: *p}, nil
}

func (db *noneDbProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
