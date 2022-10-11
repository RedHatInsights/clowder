package serviceaccount

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "serviceaccount"

// GetEnd returns the correct end provider.
func GetServiceAccount(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewServiceAccountProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetServiceAccount, 97, ProvName)
}
