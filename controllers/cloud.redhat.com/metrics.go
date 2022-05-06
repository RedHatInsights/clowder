package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var managedApps = map[string]bool{}
var managedEnvironments = map[string]bool{}
var presentApps = map[string]bool{}
var presentEnvironments = map[string]bool{}

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
	presentAppsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "clowd_app_present_apps",
			Help: "ClowdApp Present Apps",
		},
	)
	presentEnvsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "clowd_env_present_envs",
			Help: "ClowdEnv Present Envs",
		},
	)
	clowderVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clowder_version",
			Help: "ClowderVersion 1 if present, 0 if not",
		},
		[]string{"version"},
	)
	providerMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "clowder_provider_runtime",
			Help: "Provider runtime",
		},
		[]string{"provider", "source"},
	)
	requestMetrics = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "clowder_reconcile_requests",
			Help: "Clowder reconciliation requests",
		},
		[]string{"type", "name"},
	)
	reconciliationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "clowder_app_reconciliation_time",
			Help: "Clowder App reconciliation time",
		},
		[]string{"app", "env"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		managedAppsMetric,
		managedEnvsMetric,
		clowderVersion,
		providerMetrics,
		requestMetrics,
		presentAppsMetric,
		presentEnvsMetric,
		reconciliationMetrics,
	)
}
