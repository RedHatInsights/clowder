package web

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

// ProvName sets the provider name identifier
var ProvName = "web"

// CoreService is the service for the apps deployments.
var CoreService = providers.NewMultiResourceIdent(ProvName, "core_service", &core.Service{})

// WebKeycloakDeployment is the mocked keycloak deployment
var WebKeycloakDeployment = providers.NewSingleResourceIdent(ProvName, "web_keycloak_deployment", &apps.Deployment{})

// WebKeycloakService is the mocked keycloak deployment
var WebKeycloakService = providers.NewSingleResourceIdent(ProvName, "web_keycloak_service", &core.Service{})

// WebBOPDeployment is the mocked bop deployment
var WebBOPDeployment = providers.NewSingleResourceIdent(ProvName, "web_bop_deployment", &apps.Deployment{})

// WebKeycloakService is the mocked keycloak deployment
var WebBOPService = providers.NewSingleResourceIdent(ProvName, "web_bop_service", &core.Service{})

// WebSecret is the mocked secret config
var WebSecret = providers.NewSingleResourceIdent(ProvName, "web_secret", &core.Secret{})

// GetEnd returns the correct end provider.
func GetWeb(c *providers.Provider) (providers.ClowderProvider, error) {

	webMode := c.Env.Spec.Providers.Web.Mode
	switch webMode {
	case "none", "operator":
		return NewWebProvider(c)
	case "local":
		return NewLocalWebProvider(c)
	default:
		errStr := fmt.Sprintf("No matching web mode for %s", webMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	providers.ProvidersRegistration.Register(GetWeb, 1, ProvName)
}
