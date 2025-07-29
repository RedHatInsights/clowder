package confighash

import (
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	cronjobProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/cronjob"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type confighashProvider struct {
	p.Provider
}

// CoreConfigSecret is the config that is presented as the cdappconfig.json file.
var CoreConfigSecret = rc.NewSingleResourceIdent(ProvName, "core_config_secret", &core.Secret{})

// NewConfigHashProvider returns a new End provider run at the end of the provider set.
func NewConfigHashProvider(p *p.Provider) (p.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(CoreConfigSecret)
	return &confighashProvider{Provider: *p}, nil
}

func (ch *confighashProvider) EnvProvide() error {
	return nil
}

func (ch *confighashProvider) Provide(app *crd.ClowdApp) error {

	hash, err := ch.persistConfig(app)

	if err != nil {
		return err
	}

	dList := apps.DeploymentList{}
	if err := ch.Cache.List(deployProvider.CoreDeployment, &dList); err != nil {
		return err
	}

	for i := range dList.Items {
		depInner := &dList.Items[i]
		annotations := map[string]string{"configHash": hash}
		utils.UpdateAnnotations(&depInner.Spec.Template, annotations)

		if err := ch.Cache.Update(deployProvider.CoreDeployment, depInner); err != nil {
			return err
		}
	}

	jList := batch.CronJobList{}
	if err := ch.Cache.List(cronjobProvider.CoreCronJob, &jList); err != nil {
		return err
	}

	for i := range jList.Items {
		jobInner := &jList.Items[i]
		annotations := map[string]string{"configHash": hash}
		utils.UpdateAnnotations(&jobInner.Spec.JobTemplate.Spec.Template, annotations)

		if err := ch.Cache.Update(cronjobProvider.CoreCronJob, jobInner); err != nil {
			return err
		}
	}

	return nil
}
