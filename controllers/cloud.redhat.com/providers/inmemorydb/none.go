package inmemorydb

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneInMemoryDbProvider struct {
	providers.Provider
}

// NewNoneInMemoryDb returns a new none in-memory DB provider object.
func NewNoneInMemoryDb(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneInMemoryDbProvider{Provider: *p}, nil
}

func (r *noneInMemoryDbProvider) EnvProvide() error {
	return nil
}

func (r *noneInMemoryDbProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
