package metrics

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneMetricsProvider struct {
	providers.Provider
}

func NewNoneMetricsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneMetricsProvider{Provider: *p}, nil
}

func (m *noneMetricsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if !app.PreHookDone() {
		return nil
	}

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, c); err != nil {
		return err
	}

	return nil
}
