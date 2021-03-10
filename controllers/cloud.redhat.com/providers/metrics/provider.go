package metrics

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "metrics"

// GetEnd returns the correct end provider.
func GetMetrics(c *p.Provider) (p.ClowderProvider, error) {
	return NewMetricsProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetMetrics, 1, ProvName)
}
