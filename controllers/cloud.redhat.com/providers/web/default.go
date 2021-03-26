package web

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
)

type webProvider struct {
	p.Provider
}

func NewWebProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &webProvider{Provider: *p}, nil
}

func (web *webProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	c.WebPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	c.PublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	privatePort := web.Env.Spec.Providers.Web.PrivatePort
	if privatePort == 0 {
		privatePort = 10000
	}
	c.PrivatePort = utils.IntPtr(int(privatePort))

	for _, deployment := range app.Spec.Deployments {

		if err := web.makeService(&deployment, app); err != nil {
			return err
		}
	}
	return nil
}
