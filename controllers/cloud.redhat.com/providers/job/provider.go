package job

import (
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

var ProvName = "job"

func GetJob(c *p.Provider) (p.ClowderProvider, error) {
	return NewJobProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetJob, 5, ProvName)
}
