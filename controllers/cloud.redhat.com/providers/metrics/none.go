package metrics

import (
	"fmt"

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

func (m *noneMetricsProvider) EnvProvide() error {
	return nil
}

func (m *noneMetricsProvider) Provide(app *crd.ClowdApp) error {

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, m.Config); err != nil {
		return err
	}

	// Populate prometheus gateway configuration if enabled
	if m.Env.Spec.Providers.Metrics.PrometheusGateway.Deploy {
		m.Config.PrometheusGateway = &config.PrometheusGatewayConfig{
			Hostname: fmt.Sprintf("%s-prometheus-gateway.%s.svc", m.Env.Name, m.Env.Status.TargetNamespace),
			Port:     9091,
		}
	}

	return nil
}
