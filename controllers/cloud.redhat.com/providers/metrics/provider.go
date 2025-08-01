package metrics

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
)

// MetricsServiceMonitor represents the resource identifier for metrics service monitors
var MetricsServiceMonitor = rc.NewMultiResourceIdent(ProvName, "metrics-service-monitor", &prom.ServiceMonitor{})

// ProvName sets the provider name identifier
var ProvName = "metrics"

// GetMetrics returns the correct metrics provider.
func GetMetrics(c *providers.Provider) (providers.ClowderProvider, error) {
	c.Cache.AddPossibleGVKFromIdent(
		MetricsServiceMonitor,
	)
	metricsMode := c.Env.Spec.Providers.Metrics.Mode
	switch metricsMode {
	case "none", "":
		return NewNoneMetricsProvider(c)
	case "operator":
		return NewMetricsProvider(c)
	case "app-interface":
		return NewAppInterfaceMetrics(c)
	default:
		errStr := fmt.Sprintf("No matching metrics mode for %s", metricsMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	providers.ProvidersRegistration.Register(GetMetrics, 2, ProvName)
}
