package testing

import (
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

var ProvName = "testing"

// GetTestingProvider returns the iqe details for a pod
func GetTestingProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return NewTestingProvider(p)
}

func init() {
	providers.ProvidersRegistration.Register(GetTestingProvider, 1, ProvName)
}
