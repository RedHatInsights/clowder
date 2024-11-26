package deployment

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	apps "k8s.io/api/apps/v1"
	"fmt"
)

type deploymentProvider struct {
	providers.Provider
}

// CoreDeployment is the deployment for the apps deployments.
var CoreDeployment = rc.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})

func NewDeploymentProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(CoreDeployment)
	return &deploymentProvider{Provider: *p}, nil
}

func (dp *deploymentProvider) EnvProvide() error {
	return nil
}

func (dp *deploymentProvider) Provide(app *crd.ClowdApp) error {
	for _, deployment := range app.Spec.Deployments {
		fmt.Printf("deployment provider checking deployment %s", deployment.Name)
		if !deployment.UseStatefulSet {
			fmt.Printf("deployment provider processing deployment %s", deployment.Name)
			if err := dp.makeDeployment(deployment, app); err != nil {
				return err
			}
		}	
	}
	return nil
}
