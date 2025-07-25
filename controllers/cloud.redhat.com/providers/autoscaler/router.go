package autoscaler

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// autoScaleProviderRouter is a wrapper for the different autoscaler providers.
type autoScaleProviderRouter struct {
	providers.Provider
}

func NewAutoScaleProviderRouter(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		SimpleAutoScaler,
		CoreAutoScaler,
	)
	return &autoScaleProviderRouter{Provider: *p}, nil
}

func (asp *autoScaleProviderRouter) EnvProvide() error {
	return nil
}

func (asp *autoScaleProviderRouter) Provide(app *crd.ClowdApp) error {
	var err error
	for i := range app.Spec.Deployments {
		deployment := &app.Spec.Deployments[i]
		// If we find a SimpleAutoScaler config create one
		if deployment.AutoScalerSimple != nil {
			err = ProvideSimpleAutoScaler(app, asp.GetConfig(), &asp.Provider, deployment)
			continue
		}
		// If we find a Keda autoscaler config create one
		if deployment.AutoScaler != nil {
			err = ProvideKedaAutoScaler(app, asp.GetConfig(), &asp.Provider, deployment)
			continue
		}
	}
	return err
}
