package metrics

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type metricsProvider struct {
	providers.Provider
}

func NewMetricsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &metricsProvider{Provider: *p}, nil
}

func (m *metricsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	c.MetricsPort = int(m.Env.Spec.Providers.Metrics.Port)
	c.MetricsPath = m.Env.Spec.Providers.Metrics.Path

	for _, deployment := range app.Spec.Deployments {

		if err := m.makeMetrics(&deployment, app); err != nil {
			return err
		}
	}
	return nil
}
