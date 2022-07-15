package autoscaler

import (
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "autoscaler"

const ENABLED = "enabled"
const KEDA = "keda"

// GetAutoscaler returns the correct end provider.
func GetAutoScaler(c *p.Provider) (p.ClowderProvider, error) {
	mode := c.Env.Spec.Providers.AutoScaler.Mode
	//Keda is preserved as a synonym of enabled for backwards compatibility
	if mode == ENABLED || mode == KEDA {
		return NewAutoScaleProviderRouter(c)
	}
	return NewNoneAutoScalerProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetAutoScaler, 10, ProvName)
}
