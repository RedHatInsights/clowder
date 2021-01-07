package objectstore

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneObjectStoreProvider struct {
	p.Provider
	Config config.DatabaseConfig
}

func NewNoneObjectStore(p *p.Provider) (providers.ClowderProvider, error) {
	return &noneObjectStoreProvider{Provider: *p}, nil
}

func (k *noneObjectStoreProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
