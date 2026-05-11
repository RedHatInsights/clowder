package web

import (
	"fmt"

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
	web.Config.WebPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port)) // nolint:staticcheck  // ignore SA1019, we know this is deprecated
	web.Config.PublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	privatePort := web.Env.Spec.Providers.Web.PrivatePort
	if privatePort == 0 {
		privatePort = 10000
	}
	web.Config.PrivatePort = utils.IntPtr(int(privatePort))

	// Set H2C ports if configured
	if web.Env.Spec.Providers.Web.H2CPort != 0 {
		web.Config.H2CPublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.H2CPort))
	}
	h2cPrivatePort := web.Env.Spec.Providers.Web.H2CPrivatePort
	if h2cPrivatePort != 0 {
		web.Config.H2CPrivatePort = utils.IntPtr(int(h2cPrivatePort))
	}

	envTLSConfig := &web.Env.Spec.Providers.Web.TLS

	for _, deployment := range app.Spec.Deployments {
		innerDeployment := deployment
		if err := makeService(web.Cache, &innerDeployment, app, web.Env); err != nil {
			return errors.Wrap("making service", err)
		}

		// mount CA cert volume on Deployments if TLS is configured in the environment
		// (whether it is globally enabled or not, we will always mount the volume)
		if provutils.IsTLSConfiguredForEnv(envTLSConfig) {
			d := &apps.Deployment{}
			dnn := app.GetDeploymentNamespacedName(&innerDeployment)

			if err := web.Cache.Get(provDeploy.CoreDeployment, d, dnn); err != nil {
				return errors.Wrap("getting core deployment", err)
			}

			// Get CA info from app spec
			caSecretName, caFileName := web.resolveCAForApp(app)
			provutils.AddCertVolumeWithCA(&d.Spec.Template.Spec, dnn.Name, caSecretName, caFileName)

			if err := web.Cache.Update(provDeploy.CoreDeployment, d); err != nil {
				return errors.Wrap("updating core deployment", err)
			}
		}
	}

	// mount CA cert volume on CronJobs if TLS is configured in the environment
	// (whether it is globally enabled or not, we will always mount the volume)
	if provutils.IsTLSConfiguredForEnv(envTLSConfig) {
		d := &batch.CronJobList{}

		if err := web.Cache.List(provCronjob.CoreCronJob, d); err != nil {
			return errors.Wrap("get cronjob list", err)
		}

		// Get CA info from app spec
		caSecretName, caFileName := web.resolveCAForApp(app)

		for _, item := range d.Items {
			innerItem := item
			provutils.AddCertVolumeWithCA(&innerItem.Spec.JobTemplate.Spec.Template.Spec, innerItem.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name, caSecretName, caFileName)

			if err := web.Cache.Update(provCronjob.CoreCronJob, &innerItem); err != nil {
				return err

			}
		}
	}

	if envTLSConfig.Enabled {
		// if TLS is enabled environment-wide, set 'tlsCAPath' in the root level of cdappconfig
		web.Config.TlsCAPath = provutils.GetCACertPathForApp(app.Spec.TLSCertificateAuthorityName, app.Spec.TLSCertificateAuthoritySecretRef)
	}

	return nil
}

// resolveCAForApp determines which CA secret to mount based on app's CA configuration
// Returns (secretName, fileName)
// - ("", "service-ca.crt") for default (no CA specified)
// - ("", "") for system-trust-store (skip mounting)
// - ("{env}-ca-bundle", "{caname}.crt") for CA from environment bundle
// - ("{override-secret-name}", "ca.crt") for override secret
func (web *webProvider) resolveCAForApp(app *crd.ClowdApp) (string, string) {
	// Case 1: App uses override secret
	if app.Spec.TLSCertificateAuthoritySecretRef != nil {
		// Mount the app-managed secret with standard ca.crt key
		return app.Spec.TLSCertificateAuthoritySecretRef.Name, "ca.crt"
	}

	// Case 2: No CA specified - use default
	if app.Spec.TLSCertificateAuthorityName == nil {
		return "", "service-ca.crt"
	}

	caName := *app.Spec.TLSCertificateAuthorityName

	// Case 3: System trust store - don't mount any CA
	if caName == "system-trust-store" {
		return "", ""
	}

	// Case 4: CA from environment bundle
	bundleSecretName := fmt.Sprintf("%s-ca-bundle", web.Env.Name)
	fileName := fmt.Sprintf("%s.crt", caName)
	return bundleSecretName, fileName
}
