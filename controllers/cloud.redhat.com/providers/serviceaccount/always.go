package serviceaccount

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type serviceaccountProvider struct {
	p.Provider
}

func NewServiceAccountProvider(p *p.Provider) (p.ClowderProvider, error) {

	if err := createServiceAccount(p.Cache, CoreEnvServiceAccount, p.Env, p.Env.Spec.Providers.PullSecrets); err != nil {
		return nil, err
	}

	return &serviceaccountProvider{Provider: *p}, nil
}

func (sa *serviceaccountProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	if err := createServiceAccount(sa.Cache, CoreAppServiceAccount, app, sa.Env.Spec.Providers.PullSecrets); err != nil {
		return err
	}
	return nil
}
