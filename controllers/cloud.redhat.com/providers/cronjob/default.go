package cronjob

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type cronjobProvider struct {
	p.Provider
}

func NewCronJobProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &cronjobProvider{Provider: *p}, nil
}

func (j *cronjobProvider) EnvProvide() error {
	return nil
}

func (j *cronjobProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	for _, cronjob := range app.Spec.Jobs {
		if cronjob.Schedule != "" {
			if err := j.makeCronJob(&cronjob, app); err != nil {
				return err
			}
		}
	}
	return nil
}
