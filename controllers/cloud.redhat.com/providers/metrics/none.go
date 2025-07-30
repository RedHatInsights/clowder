package metrics

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneMetricsProvider struct {
	providers.Provider
}

// NewNoneMetricsProvider creates a new metrics provider that does nothing
func NewNoneMetricsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneMetricsProvider{Provider: *p}, nil
}

func (m *noneMetricsProvider) EnvProvide() error {
	return nil
}

func (m *noneMetricsProvider) Provide(app *crd.ClowdApp) error {

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, m.Config); err != nil {
		return err
	}

	// Note: Prometheus Gateway is not supported in none mode
	// as no metrics infrastructure is deployed. The configuration
	// is intentionally not populated here.

	return nil
}
