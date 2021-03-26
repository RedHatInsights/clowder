package deployment

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type deploymentProvider struct {
	p.Provider
}

func NewDeploymentProvider(p *p.Provider) (p.ClowderProvider, error) {
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
