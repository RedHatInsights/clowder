package serviceaccount

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
)

// ProvName sets the provider name identifier
var ProvName = "serviceaccount"

// CoreAppServiceAccount is the serviceaccount for the apps.
var CoreAppServiceAccount = p.NewSingleResourceIdent(ProvName, "core_app_service_account", &core.ServiceAccount{})

// CoreEnvServiceAccount is the serviceaccount for the env.
var CoreEnvServiceAccount = p.NewSingleResourceIdent(ProvName, "core_env_service_account", &core.ServiceAccount{})

// GetEnd returns the correct end provider.
func GetServiceAccount(c *p.Provider) (p.ClowderProvider, error) {
	return NewServiceAccountProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetServiceAccount, 1, ProvName)
}
