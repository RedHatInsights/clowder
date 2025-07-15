package autoscaler

import (
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	keda "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	v2 "k8s.io/api/autoscaling/v2"

	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "autoscaler"

const ENABLED = "enabled"
const KEDA = "keda"

// CoreAutoScaler is the config that is presented as the cdappconfig.json file.
var CoreAutoScaler = rc.NewMultiResourceIdent(ProvName, "core_autoscaler", &keda.ScaledObject{})
var SimpleAutoScaler = rc.NewMultiResourceIdent(ProvName, "simple_hpa", &v2.HorizontalPodAutoscaler{})

// GetAutoscaler returns the correct end provider.
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
