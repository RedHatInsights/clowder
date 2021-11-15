package autoscaler

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
)

type autoscalerProvider struct {
	p.Provider
}

// CoreConfigSecret is the config that is presented as the cdappconfig.json file.
var CoreAutoScaler = p.NewMultiResourceIdent(ProvName, "core_autoscaler", &keda.ScaledObject{})

// NewConfigHashProvider returns a new End provider run at the end of the provider set.
func NewAutoScalerProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &autoscalerProvider{Provider: *p}, nil
}

func (asp *autoscalerProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	for _, deployment := range app.Spec.Deployments {

		// Create the autoscaler if one is defined
		if deployment.AutoScaler != nil {

			if err := asp.makeAutoScalers(&deployment, app, c); err != nil {
				return err
			}
		}
	}
	return nil
}
