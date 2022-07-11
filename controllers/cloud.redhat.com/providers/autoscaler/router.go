package autoscaler

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

//autoScaleProviderRouter is a wrapper for the different autoscaler providers.
type autoScaleProviderRouter struct {
	providers.Provider
}

func NewAutoScaleProviderRouter(p *providers.Provider) (providers.ClowderProvider, error) {
	return &autoScaleProviderRouter{Provider: *p}, nil
}

func (asp *autoScaleProviderRouter) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	for _, deployment := range app.Spec.Deployments {
		//If we find a SimpleAutoScaler config create one
		if deployment.AutoScalerSimple != nil {
			ProvideSimpleAutoScaler(app, c, &asp.Provider, deployment)
			continue
		}
		//If we find a Keda autoscaler config create one
		if deployment.AutoScaler != nil {
			ProvideKedaAutoScaler(app, c, &asp.Provider, deployment)
			continue
		}
	}
	return nil
}
