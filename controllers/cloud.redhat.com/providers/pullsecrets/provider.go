package pullsecrets

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
)

// ProvName sets the provider name identifier
var ProvName = "pullsecret"

// CoreEnvPullSecrets is the pull_secrets for the app.
var CoreEnvPullSecrets = providers.NewMultiResourceIdent(ProvName, "core_env_pull_secrets", &core.Secret{})

// GetPullSecret returns the correct end provider.
func GetPullSecret(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewPullSecretProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetPullSecret, 98, ProvName)
}
