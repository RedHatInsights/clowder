package mock

import (
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

// ProvName sets the provider name identifier
var ProvName = "mock"

// MockKeycloakDeployment is the mocked keycloak deployment
var MockKeycloakDeployment = providers.NewSingleResourceIdent(ProvName, "mock_keycloak_deployment", &apps.Deployment{})

// MockKeycloakService is the mocked keycloak deployment
var MockKeycloakService = providers.NewSingleResourceIdent(ProvName, "mock_keycloak_service", &core.Service{})

// MockBOPDeployment is the mocked bop deployment
var MockBOPDeployment = providers.NewSingleResourceIdent(ProvName, "mock_bop_deployment", &apps.Deployment{})

// MockKeycloakService is the mocked keycloak deployment
var MockBOPService = providers.NewSingleResourceIdent(ProvName, "mock_bop_service", &core.Service{})

// MockSecret is the mocked secret config
var MockSecret = providers.NewSingleResourceIdent(ProvName, "mock_secret", &core.Secret{})

// GetMock returns the correct mock provider.
func GetMock(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewMockProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetMock, 2, ProvName)
}
