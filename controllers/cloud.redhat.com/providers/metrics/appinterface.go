package metrics

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type appinterfaceMetricsProvider struct {
	p.Provider
}

func NewAppInterfaceMetrics(p *p.Provider) (p.ClowderProvider, error) {
	return &appinterfaceMetricsProvider{Provider: *p}, nil
}

func (m *appinterfaceMetricsProvider) EnvProvide() error {
	return nil
}

func (m *appinterfaceMetricsProvider) Provide(app *crd.ClowdApp) error {

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

	if clowderconfig.LoadedConfig.Features.CreateServiceMonitor {
		if err := createServiceMonitorObjects(m.Cache, m.Env, app, "app-sre", "openshift-customer-monitoring"); err != nil {
			return err
		}
	}
	return nil
}
