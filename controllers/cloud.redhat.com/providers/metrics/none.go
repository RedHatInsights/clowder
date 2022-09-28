package metrics

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
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

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, m.Config.Config); err != nil {
		return err
	}

	return nil
}
