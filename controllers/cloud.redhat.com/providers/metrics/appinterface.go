package metrics

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
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

	// Note: Prometheus Gateway is not supported in app-interface mode
	// as it requires operator-managed resources. The configuration
	// is intentionally not populated here.

	if clowderconfig.LoadedConfig.Features.CreateServiceMonitor {
		if err := createServiceMonitorObjects(m.Cache, m.Env, app, "app-sre", "openshift-customer-monitoring"); err != nil {
			return err
		}
	}
	return nil
}
