package dependencies

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "dependencies"

// GetDependencies returns the correct end provider.
func GetDependencies(c *p.Provider) (p.ClowderProvider, error) {
	return NewDependenciesProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetDependencies, 4, ProvName)
}
