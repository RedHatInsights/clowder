package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
)

func TestReconcileMetricsStartDisabled(t *testing.T) {
	assert.False(t, clowderconfig.LoadedConfig.Features.ReconciliationMetrics)
	reconciler := ReconciliationMetrics{}
	reconciler.init("TestApp", "TestEnv")
	assert.Empty(t, reconciler.reconcileStartTime)
	reconciler.start()
	assert.Empty(t, reconciler.reconcileStartTime)
}

func TestReconcileMetricsStartEnabled(t *testing.T) {
	clowderconfig.LoadedConfig.Features.ReconciliationMetrics = true
	reconciler := ReconciliationMetrics{}
	reconciler.init("TestApp", "TestEnv")
	assert.Empty(t, reconciler.reconcileStartTime)
	reconciler.start()
	assert.NotEmpty(t, reconciler.reconcileStartTime)
}

func TestReconcileMetricsEndEnabled(_ *testing.T) {
	// After hours of messing with the prometheus lib I'm convinced there's no way
	// to read metrics from it. Metrics go into it, but you don't get metrics out of it
	// I originally wanted to assert on some valid equivalent of reconciliationMetrics is empty
	// before reconcileMetricsEnd and then not empty after but I don't see a way to do that
	// This test will catch errors in the underlying code, but it asserts nothing :(
	clowderconfig.LoadedConfig.Features.ReconciliationMetrics = true
	reconciler := ReconciliationMetrics{}
	reconciler.init("TestApp", "TestEnv")
	reconciler.start()
	reconciler.stop()
}
