package autoscaler

import (
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "autoscaler"

// GetAutoscaler returns the correct end provider.
func GetAutoScaler(c *p.Provider) (p.ClowderProvider, error) {
	return NewAutoScalerProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetAutoScaler, 10, ProvName)
}
