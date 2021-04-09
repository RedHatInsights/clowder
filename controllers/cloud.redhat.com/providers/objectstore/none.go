package objectstore

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneObjectStoreProvider struct {
	providers.Provider
}

// NewNoneObjectStore returns a new none object store provider object.
func NewNoneObjectStore(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneObjectStoreProvider{Provider: *p}, nil
}

func (k *noneObjectStoreProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
