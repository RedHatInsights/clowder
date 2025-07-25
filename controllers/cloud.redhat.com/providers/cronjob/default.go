package cronjob

import (
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	batch "k8s.io/api/batch/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
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

	for i := range app.Spec.Jobs {
		innerCronjob := &app.Spec.Jobs[i]
		if innerCronjob.Schedule != "" && !innerCronjob.Disabled {
			if err := j.makeCronJob(innerCronjob, app); err != nil {
				return err
			}
		}
	}
	return nil
}
