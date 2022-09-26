package objectstore

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneObjectStoreProvider struct {
	providers.Provider
}

// NewNoneObjectStore returns a new none object store provider object.
func NewNoneObjectStore(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneObjectStoreProvider{Provider: *p}, nil
}

func (k *noneObjectStoreProvider) EnvProvide() error {
	return nil
}

func (k *noneObjectStoreProvider) Provide(app *crd.ClowdApp) error {
	return nil
}
