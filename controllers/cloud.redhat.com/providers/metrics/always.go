package metrics

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type metricsProvider struct {
	p.Provider
}

func NewMetricsProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &metricsProvider{Provider: *p}, nil
}

func (m *metricsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	for _, deployment := range app.Spec.Deployments {

		if err := m.makeMetrics(&deployment, app); err != nil {
			return err
		}
	}
	return nil
}
