package certificateauthority

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "certificateauthority"

// GetCertificateAuthority returns the correct certificate authority provider.
func GetCertificateAuthority(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewCertificateAuthorityProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetCertificateAuthority, 2, ProvName)
}
