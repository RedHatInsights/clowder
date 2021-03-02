package inmemorydb

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneInMemoryDbProvider struct {
	p.Provider
	Config config.DatabaseConfig
}

// NewNoneInMemoryDb returns a new none in-memory DB provider object.
func NewNoneInMemoryDb(p *p.Provider) (providers.ClowderProvider, error) {
	return &noneInMemoryDbProvider{Provider: *p}, nil
}

func (r *noneInMemoryDbProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
