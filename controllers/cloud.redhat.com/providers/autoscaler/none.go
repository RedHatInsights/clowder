package autoscaler

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneAutoScalerProvider struct {
	providers.Provider
}

// NewNoneAutoScalerProvider returns a new none autoscaler provider object.
func NewNoneAutoScalerProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneAutoScalerProvider{Provider: *p}, nil
}

func (db *noneAutoScalerProvider) EnvProvide() error {
	return nil
}

func (db *noneAutoScalerProvider) Provide(_ *crd.ClowdApp) error {
	return nil
}
