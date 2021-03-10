package deployment

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
)

// ProvName sets the provider name identifier
var ProvName = "deployment"

// CoreDeployment is the deployment for the apps deployments.
var CoreDeployment = p.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})

// GetEnd returns the correct end provider.
func GetDeployment(c *p.Provider) (p.ClowderProvider, error) {
	return NewDeploymentProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetDeployment, 0, ProvName)
}
