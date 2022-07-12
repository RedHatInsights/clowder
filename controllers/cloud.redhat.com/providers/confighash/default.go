package confighash

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	cronjobProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/cronjob"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

type confighashProvider struct {
	p.Provider
}

// CoreConfigSecret is the config that is presented as the cdappconfig.json file.
var CoreConfigSecret = rc.NewSingleResourceIdent(ProvName, "core_config_secret", &core.Secret{})

// NewConfigHashProvider returns a new End provider run at the end of the provider set.
func NewConfigHashProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &confighashProvider{Provider: *p}, nil
}

func (ch *confighashProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	hash, err := ch.persistConfig(app, c)

	if err != nil {
		return err
	}

	dList := apps.DeploymentList{}
	if err := ch.Cache.List(deployProvider.CoreDeployment, &dList); err != nil {
		return err
	}

	for _, deployment := range dList.Items {
		annotations := map[string]string{"configHash": hash}
		utils.UpdateAnnotations(&deployment.Spec.Template, annotations)

		ch.Cache.Update(deployProvider.CoreDeployment, &deployment)
	}

	jList := batch.CronJobList{}
	if err := ch.Cache.List(cronjobProvider.CoreCronJob, &jList); err != nil {
		return err
	}

	for _, job := range jList.Items {
		annotations := map[string]string{"configHash": hash}
		utils.UpdateAnnotations(&job.Spec.JobTemplate.Spec.Template, annotations)

		ch.Cache.Update(cronjobProvider.CoreCronJob, &job)
	}

	return nil
}
