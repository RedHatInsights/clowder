package web

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type webProvider struct {
	providers.Provider
}

func NewWebProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CoreService,
	)
	return &webProvider{Provider: *p}, nil
}

func (web *webProvider) EnvProvide() error {
	return nil
}

func (web *webProvider) Provide(app *crd.ClowdApp) error {

	web.Config.WebPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	web.Config.PublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	privatePort := web.Env.Spec.Providers.Web.PrivatePort
	if privatePort == 0 {
		privatePort = 10000
	}
	web.Config.PrivatePort = utils.IntPtr(int(privatePort))

	for _, deployment := range app.Spec.Deployments {
		d := deployment
		if err := makeService(web.Cache, &d, app, web.Env); err != nil {
			return err
		}
	}
	return nil
}
