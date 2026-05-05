package reverseproxy

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneReverseProxyProvider struct {
	providers.Provider
}

// NewNoneReverseProxy returns a no-op reverse proxy provider.
func NewNoneReverseProxy(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneReverseProxyProvider{Provider: *p}, nil
}

func (n *noneReverseProxyProvider) EnvProvide() error {
	return nil
}

func (n *noneReverseProxyProvider) Provide(_ *crd.ClowdApp) error {
	return nil
}
