package serviceaccount

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

// ProvName sets the provider name identifier
var ProvName = "serviceaccount"

// CoreDeploymentRoleBinding is the rolebinding for the apps.
var CoreDeploymentRoleBinding = providers.NewMultiResourceIdent(ProvName, "core_deployment_role_binding", &rbac.RoleBinding{})

// CoreDeploymentServiceAccount is the serviceaccount for the apps.
var CoreDeploymentServiceAccount = providers.NewMultiResourceIdent(ProvName, "core_deployment_service_account", &core.ServiceAccount{})

// CoreAppServiceAccount is the serviceaccount for the apps.
var CoreAppServiceAccount = providers.NewSingleResourceIdent(ProvName, "core_app_service_account", &core.ServiceAccount{})

// CoreEnvServiceAccount is the serviceaccount for the env.
var CoreEnvServiceAccount = providers.NewSingleResourceIdent(ProvName, "core_env_service_account", &core.ServiceAccount{})

// IQEServiceAccount is the serviceaccount for the iqe testing.
var IQEServiceAccount = providers.NewMultiResourceIdent(ProvName, "iqe_service_account", &core.ServiceAccount{})

// IQERoleBinding is the reolbinding for the env.
var IQERoleBinding = providers.NewMultiResourceIdent(ProvName, "iqe_role_binding", &rbac.RoleBinding{})

// GetEnd returns the correct end provider.
func GetServiceAccount(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewServiceAccountProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetServiceAccount, 97, ProvName)
}
