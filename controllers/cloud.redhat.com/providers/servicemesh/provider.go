package servicemesh

import (
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "servicemesh"

// GetServiceMesh returns the correct end provider.
func GetServiceMesh(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewServiceMeshProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetServiceMesh, 98, ProvName)
}
