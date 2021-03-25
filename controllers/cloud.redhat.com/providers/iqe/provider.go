package iqe

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// GetIqe returns the iqe details for a pod
func GetIqe(c *p.Provider) (p.ClowderProvider, error) {
	return NewIqeProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetIqe, 1, "iqe")
}
