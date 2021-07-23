package web

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
)

// ProvName sets the provider name identifier
var ProvName = "web"

// CoreService is the service for the apps deployments.
var CoreService = providers.NewMultiResourceIdent(ProvName, "core_service", &core.Service{})

// GetEnd returns the correct end provider.
func GetWeb(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewWebProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetWeb, 1, ProvName)
}
