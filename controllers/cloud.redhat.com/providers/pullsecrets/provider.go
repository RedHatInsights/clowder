package pullsecrets

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "pullsecret"

// GetPullSecret returns the correct end provider.
func GetPullSecret(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewPullSecretProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetPullSecret, 98, ProvName)
}
