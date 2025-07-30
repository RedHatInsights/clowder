// Package deployment provides deployment management functionality for Clowder applications
package deployment

import (
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	apps "k8s.io/api/apps/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type deploymentProvider struct {
	providers.Provider
}

// CoreDeployment is the deployment for the apps deployments.
var CoreDeployment = rc.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})

// NewDeploymentProvider creates a new deployment provider instance
func NewDeploymentProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(CoreDeployment)
	return &deploymentProvider{Provider: *p}, nil
}

func (dp *deploymentProvider) EnvProvide() error {
	return nil
}

func (dp *deploymentProvider) Provide(app *crd.ClowdApp) error {

	for i := range app.Spec.Deployments {
		deployment := &app.Spec.Deployments[i]
		if err := dp.makeDeployment(deployment, app); err != nil {
			return err
		}
	}
	return nil
}
