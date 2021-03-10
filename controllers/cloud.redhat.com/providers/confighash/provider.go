package confighash

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "confighash"

// GetConfigHash returns the correct end provider.
func GetConfigHash(c *p.Provider) (p.ClowderProvider, error) {
	return NewConfigHashProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetConfigHash, 99, ProvName)
}
