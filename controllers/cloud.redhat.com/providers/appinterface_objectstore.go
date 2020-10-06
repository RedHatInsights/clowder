package providers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
)

type AppInterfaceObjectstoreProvider struct {
	Provider
	Config config.ObjectStoreConfig
}

func (a *AppInterfaceObjectstoreProvider) Configure(c *config.AppConfig) {
	c.ObjectStore = &a.Config
}

func NewAppInterfaceObjectstore(p *Provider) (ObjectStoreProvider, error) {
	provider := AppInterfaceObjectstoreProvider{Provider: *p}

	return &provider, nil
}

func (a *AppInterfaceObjectstoreProvider) CreateBuckets(app *crd.ClowdApp) error {
	return nil
}
