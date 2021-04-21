package iqe

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

var ProvName = "testing"

// GetTestingProvider returns the iqe details for a pod
func GetTestingProvider(c *p.Provider) (p.ClowderProvider, error) {
	return NewTestingProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetTestingProvider, 1, ProvName)
}
