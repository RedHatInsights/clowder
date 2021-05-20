package metrics

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneMetricsProvider struct {
	providers.Provider
}

func NewNoneMetricsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneMetricsProvider{Provider: *p}, nil
}

func (m *noneMetricsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, c); err != nil {
		return err
	}

	return nil
}
