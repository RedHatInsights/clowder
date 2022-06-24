package sidecar

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

var IMAGE_SIDECAR_TOKEN_REFRESHER = "quay.io/observatorium/token-refresher:master-2021-02-05-5da9663"

// ProvName sets the provider name identifier
var ProvName = "sidecar"

// GetEnd returns the correct end provider.
func GetSideCar(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewSidecarProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetSideCar, 98, ProvName)
}
