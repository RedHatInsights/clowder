package web

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type webProvider struct {
	p.Provider
}

func NewWebProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &webProvider{Provider: *p}, nil
}

func (web *webProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	for _, deployment := range app.Spec.Deployments {

		if err := web.makeService(&deployment, app); err != nil {
			return err
		}
	}
	return nil
}
