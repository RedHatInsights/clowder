package namespace

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "namespace"

// GetNamespaceProvider returns the correct end provider.
func GetNamespaceProvider(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewNamespaceProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetNamespaceProvider, 50, ProvName)
}
