package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var managedApps = map[string]bool{}
var managedEnvironments = map[string]bool{}

var (
	managedAppsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "clowd_app_managed_apps",
			Help: "ClowdApp Managed Apps",
		},
	)
	managedEnvsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "clowd_env_managed_envs",
			Help: "ClowdEnv Managed Envs",
		},
	)
	clientOpsMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "clowder_api_calls",
			Help: "Number of calls to API",
		},
		[]string{"operation"},
	)
	clowderVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clowder_version",
			Help: "ClowderVersion 1 if present, 0 if not",
		},
		[]string{"version"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(managedAppsMetric, managedEnvsMetric, clientOpsMetric, clowderVersion)
}
