package deployment

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

// ProvName sets the provider name identifier
var ProvName = "deployment"

// CoreDeployment is the deployment for the apps deployments.
var CoreDeployment = rc.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})

// GetEnd returns the correct end provider.
func GetDeployment(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewDeploymentProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetDeployment, 0, ProvName)
}
