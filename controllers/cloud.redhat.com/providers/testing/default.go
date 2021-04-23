package testing

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type testingProvider struct {
	providers.Provider
	Config config.TestingConfig
}

func NewTestingProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &testingProvider{Provider: *p, Config: config.TestingConfig{}}, nil
}

func (t *testingProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if err := t.MakeTestingConfig(app, c); err != nil {
		return err
	}
	return nil
}
