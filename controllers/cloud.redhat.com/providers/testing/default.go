package iqe

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
	testing := &testingProvider{Provider: *p, Config: config.TestingConfig{}}

	testingSettings := p.Env.Spec.Providers.Testing

	iqeSettings := p.Env.Spec.Providers.Testing.Iqe
	testing.Config = config.TestingConfig{
		K8SAccessLevel: string(testingSettings.K8SAccessLevel),
		ConfigAccess:   string(testingSettings.ConfigAccess),
		Iqe: &config.IqeConfig{
			ImageBase: iqeSettings.ImageBase,
		},
	}

	return testing, nil
}

func (tp *testingProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	c.Testing = &tp.Config
	return nil
}
