package cronjob

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	batch "k8s.io/api/batch/v1"
)

type cronjobProvider struct {
	p.Provider
}

// CoreCronJob is the cronjob for the apps cronjobs.
var CoreCronJob = rc.NewMultiResourceIdent(ProvName, "core_cronjob", &batch.CronJob{})

func NewCronJobProvider(p *p.Provider) (p.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(CoreCronJob)
	return &cronjobProvider{Provider: *p}, nil
}

func (j *cronjobProvider) EnvProvide() error {
	return nil
}

func (j *cronjobProvider) Provide(app *crd.ClowdApp) error {

	for _, cronjob := range app.Spec.Jobs {
		cj := cronjob
		if cj.Schedule != "" {
			if err := j.makeCronJob(&cj, app); err != nil {
				return err
			}
		}
	}
	return nil
}
