package metrics

import (
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

func (m *appinterfaceMetricsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, c); err != nil {
		return err
	}

	if clowderconfig.LoadedConfig.Features.CreateServiceMonitor {
		if err := createServiceMonitorObjects(m.Cache, m.Env, app, c, "app-sre", "openshift-customer-monitoring"); err != nil {
			return err
		}
	}
	return nil
}
