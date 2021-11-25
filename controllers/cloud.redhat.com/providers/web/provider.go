package web

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

// ProvName sets the provider name identifier
var ProvName = "web"

// CoreService is the service for the apps deployments.
var CoreService = rc.NewMultiResourceIdent(ProvName, "core_service", &core.Service{})

// WebKeycloakDeployment is the mocked keycloak deployment
var WebKeycloakDeployment = rc.NewSingleResourceIdent(ProvName, "web_keycloak_deployment", &apps.Deployment{}, rc.ResourceOptions{WriteNow: true})

// WebKeycloakService is the mocked keycloak deployment
var WebKeycloakService = rc.NewSingleResourceIdent(ProvName, "web_keycloak_service", &core.Service{}, rc.ResourceOptions{WriteNow: true})

// WebKeycloakIngress is the keycloak ingress
var WebKeycloakIngress = rc.NewSingleResourceIdent(ProvName, "web_keycloak_ingress", &networking.Ingress{})

// WebBOPDeployment is the mocked bop deployment
var WebBOPDeployment = rc.NewSingleResourceIdent(ProvName, "web_bop_deployment", &apps.Deployment{})

// WebKeycloakService is the mocked keycloak deployment
var WebBOPService = rc.NewSingleResourceIdent(ProvName, "web_bop_service", &core.Service{})

// WebKeycloakIngress is the mocked bop ingress
var WebBOPIngress = rc.NewSingleResourceIdent(ProvName, "web_bop_ingress", &networking.Ingress{})

// WebSecret is the mocked secret config
var WebSecret = rc.NewMultiResourceIdent(ProvName, "web_secret", &core.Secret{})

// WebKeycloakSecret is the mocked secret config
var WebKeycloakSecret = rc.NewSingleResourceIdent(ProvName, "web_keycloak_secret", &core.Secret{}, rc.ResourceOptions{WriteNow: true})

// WebIngress is the mocked secret config
var WebIngress = rc.NewMultiResourceIdent(ProvName, "web_ingress", &networking.Ingress{})

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
