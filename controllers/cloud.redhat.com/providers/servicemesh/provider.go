package servicemesh

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "servicemesh"

// GetServiceMesh returns the correct end provider.
func GetServiceMesh(c *p.Provider) (p.ClowderProvider, error) {
	return NewServiceMeshProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetServiceMesh, 98, ProvName)
}
