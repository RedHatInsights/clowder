package sidecar

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

var DefaultImageSideCarTokenRefresher = "quay.io/observatorium/token-refresher:master-2023-09-20-f5e3403" // nolint:gosec

// ProvName sets the provider name identifier
var ProvName = "sidecar"

// GetEnd returns the correct end provider.
func GetSideCar(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewSidecarProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetSideCar, 98, ProvName)
}
