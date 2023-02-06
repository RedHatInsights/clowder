package web

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provDeploy "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type webProvider struct {
	providers.Provider
}

func NewWebProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CoreService,
		CoreEnvoyConfigMap,
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

	if err := web.populateCA(); err != nil {
		return err
	}

	for _, deployment := range app.Spec.Deployments {
		innerDeployment := deployment
		if err := makeService(web.Ctx, web.Client, web.Cache, &innerDeployment, app, web.Env); err != nil {
			return err
		}

		if web.Env.Spec.Providers.Web.TLS.Enabled {
			d := &apps.Deployment{}
			dnn := app.GetDeploymentNamespacedName(&innerDeployment)

			if err := web.Cache.Get(provDeploy.CoreDeployment, d, dnn); err != nil {
				return err
			}

			addCertVolume(d, dnn.Name)

			if err := web.Cache.Update(provDeploy.CoreDeployment, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func (web *webProvider) populateCA() error {
	if web.Env.Spec.Providers.Web.TLS.Enabled {
		web.Config.TLSPath = utils.StringPtr("/cdapp/certs/service-ca.crt")
	}
	return nil
}
