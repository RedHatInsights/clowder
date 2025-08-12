package autoscaler

import (
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	keda "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	v2 "k8s.io/api/autoscaling/v2"

	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "autoscaler"

// ENABLED represents the enabled mode for the autoscaler provider
const ENABLED = "enabled"

// KEDA represents the KEDA mode for the autoscaler provider (synonym for enabled)
const KEDA = "keda"

// CoreAutoScaler is the config that is presented as the cdappconfig.json file.
var CoreAutoScaler = rc.NewMultiResourceIdent(ProvName, "core_autoscaler", &keda.ScaledObject{})

// SimpleAutoScaler represents the resource identifier for simple HPA autoscaling
var SimpleAutoScaler = rc.NewMultiResourceIdent(ProvName, "simple_hpa", &v2.HorizontalPodAutoscaler{})

// GetAutoScaler returns the correct autoscaler provider.
func GetAutoScaler(c *p.Provider) (p.ClowderProvider, error) {
	mode := c.Env.Spec.Providers.AutoScaler.Mode
	// Keda is preserved as a synonym of enabled for backwards compatibility
	if mode == ENABLED || mode == KEDA {
		return NewAutoScaleProviderRouter(c)
	}
	return NewNoneAutoScalerProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetAutoScaler, 10, ProvName)
}
