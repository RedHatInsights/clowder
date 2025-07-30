package deployment

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "deployment"

// GetDeployment returns the correct deployment provider.
func GetDeployment(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewDeploymentProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetDeployment, 0, ProvName)
}
