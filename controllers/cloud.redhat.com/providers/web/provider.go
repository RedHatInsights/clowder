package web

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
)

// ProvName sets the provider name identifier
var ProvName = "web"

// CoreService is the service for the apps deployments.
var CoreService = p.NewMultiResourceIdent(ProvName, "core_service", &core.Service{})

// GetEnd returns the correct end provider.
func GetWeb(c *p.Provider) (p.ClowderProvider, error) {
	return NewWebProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetWeb, 1, ProvName)
}
