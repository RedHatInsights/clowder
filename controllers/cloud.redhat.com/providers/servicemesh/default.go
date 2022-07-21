package servicemesh

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type servicemeshProvider struct {
	providers.Provider
}

// NewServiceMeshProvider returns a new End provider run at the end of the provider set.
func NewServiceMeshProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &servicemeshProvider{Provider: *p}, nil
}

func (ch *servicemeshProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if ch.Env.Spec.Providers.ServiceMesh.Mode != "enabled" {
		return nil
	}

	dList := apps.DeploymentList{}
	if err := ch.Cache.List(deployProvider.CoreDeployment, &dList); err != nil {
		return err
	}

	for _, deployment := range dList.Items {
		annotations := map[string]string{
			"sidecar.istio.io/inject":                       "true",
			"traffic.sidecar.istio.io/excludeOutboundPorts": "443,9093,5432,10000",
		}
		utils.UpdateAnnotations(&deployment.Spec.Template, annotations)

		ch.Cache.Update(deployProvider.CoreDeployment, &deployment)
	}

	return nil
}
