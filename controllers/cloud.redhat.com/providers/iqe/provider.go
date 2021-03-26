package iqe

import (
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
)

var ProvName = "iqe"

var IqeSecret = p.NewSingleResourceIdent(ProvName, "iqe", &core.Secret{})

// GetIqe returns the iqe details for a pod
func GetIqe(c *p.Provider) (p.ClowderProvider, error) {
	return NewIqeProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetIqe, 1, "iqe")
}
