package cronjob

import (
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "cronjob"

// GetCronJob returns the correct cronjob provider.
func GetCronJob(c *p.Provider) (p.ClowderProvider, error) {
	return NewCronJobProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetCronJob, 4, ProvName)
}
