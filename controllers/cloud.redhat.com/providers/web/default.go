package web

import (
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provCronjob "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/cronjob"
	provDeploy "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type webProvider struct {
	providers.Provider
}

// NewWebProvider creates a new web provider instance
func NewWebProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CoreService,
		CoreCaddyConfigMap,
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
		return errors.Wrap("populating ca", err)
	}

	for _, deployment := range app.Spec.Deployments {
		innerDeployment := deployment
		if err := makeService(web.Cache, &innerDeployment, app, web.Env); err != nil {
			return errors.Wrap("making service", err)
		}

		if web.Env.Spec.Providers.Web.TLS.Enabled {
			d := &apps.Deployment{}
			dnn := app.GetDeploymentNamespacedName(&innerDeployment)

			if err := web.Cache.Get(provDeploy.CoreDeployment, d, dnn); err != nil {
				return errors.Wrap("getting core deployment", err)
			}

			provutils.AddCertVolume(&d.Spec.Template.Spec, dnn.Name)

			if err := web.Cache.Update(provDeploy.CoreDeployment, d); err != nil {
				return errors.Wrap("updating core deployment", err)
			}
		}
	}

	if web.Env.Spec.Providers.Web.TLS.Enabled {
		d := &batch.CronJobList{}

		if err := web.Cache.List(provCronjob.CoreCronJob, d); err != nil {
			return errors.Wrap("get cronjob list", err)
		}

		for _, item := range d.Items {
			innerItem := item
			provutils.AddCertVolume(&innerItem.Spec.JobTemplate.Spec.Template.Spec, innerItem.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name)

			if err := web.Cache.Update(provCronjob.CoreCronJob, &innerItem); err != nil {
				return err

			}
		}
	}

	return nil
}

func (web *webProvider) populateCA() error {
	if web.Env.Spec.Providers.Web.TLS.Enabled {
		web.Config.TlsCAPath = utils.StringPtr("/cdapp/certs/service-ca.crt")
	}
	return nil
}
