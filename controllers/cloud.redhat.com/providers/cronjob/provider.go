package cronjob

import (
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	batch "k8s.io/api/batch/v1"
)

// ProvName sets the provider name identifier
var ProvName = "cronjob"

// CoreCronJob is the cronjob for the apps cronjobs.
var CoreCronJob = rc.NewMultiResourceIdent(ProvName, "core_cronjob", &batch.CronJob{})

// GetCronJob returns the correct cronjob provider.
func GetCronJob(c *p.Provider) (p.ClowderProvider, error) {
	return NewCronJobProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetCronJob, 4, ProvName)
}
