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
	managedAppResourceMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clowd_app_resources",
			Help: "Clowd App Resources",
		},
		[]string{"env", "app", "type"},
	)
	managedEnvResourceMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clowd_env_resources",
			Help: "Clowd Env Resources",
		},
		[]string{"env", "type"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(managedAppsMetric, managedEnvsMetric, managedAppResourceMetric, managedEnvResourceMetric)
}
