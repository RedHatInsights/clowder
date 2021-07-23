package deployment

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type deploymentProvider struct {
	providers.Provider
}

func NewDeploymentProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &deploymentProvider{Provider: *p}, nil
}

func (dp *deploymentProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	for _, deployment := range app.Spec.Deployments {

		if err := dp.makeDeployment(deployment, app); err != nil {
			return err
		}
	}
	return nil
}
