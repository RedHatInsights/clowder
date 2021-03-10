package cronjob

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	batch "k8s.io/api/batch/v1beta1"
)

// ProvName sets the provider name identifier
var ProvName = "cronjob"

// CoreCronJob is the croncronjob for the apps cronjobs.
var CoreCronJob = p.NewMultiResourceIdent(ProvName, "core_cronjob", &batch.CronJob{})

// GetEnd returns the correct end provider.
func GetCronJob(c *p.Provider) (p.ClowderProvider, error) {
	return NewCronJobProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetCronJob, 4, ProvName)
}
