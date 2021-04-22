package metrics

import (
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "metrics"

// GetEnd returns the correct end provider.
func GetMetrics(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewMetricsProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetMetrics, 2, ProvName)
}
