package serviceaccount

import (
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/database"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/featureflags"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/inmemorydb"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/kafka"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/objectstore"
	apps "k8s.io/api/apps/v1"
)

type serviceaccountProvider struct {
	providers.Provider
}

func NewServiceAccountProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	if err := createServiceAccount(p.Cache, CoreEnvServiceAccount, p.Env, p.Env.Spec.Providers.PullSecrets); err != nil {
		return nil, err
	}

	resourceIdentsToUpdate := []providers.ResourceIdent{
		featureflags.LocalFFDBDeployment,
		kafka.LocalKafkaDeployment,
		kafka.LocalZookeeperDeployment,
		objectstore.MinioDeployment,
	}

	for _, resourceIdent := range resourceIdentsToUpdate {
		if obj, ok := resourceIdent.(providers.ResourceIdentSingle); ok {
			dd := &apps.Deployment{}
			if err := p.Cache.Get(obj, dd); err != nil {
				if strings.Contains(err.Error(), "not found") {
					continue
				}
			}
			dd.Spec.Template.Spec.ServiceAccountName = p.Env.GetClowdSAName()
			if err := p.Cache.Update(obj, dd); err != nil {
				return nil, err
			}
		}
	}

	return &serviceaccountProvider{Provider: *p}, nil
}

func (sa *serviceaccountProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	if err := createServiceAccount(sa.Cache, CoreAppServiceAccount, app, sa.Env.Spec.Providers.PullSecrets); err != nil {
		return err
	}

	resourceIdentsToUpdate := []providers.ResourceIdent{
		database.LocalDBDeployment,
		inmemorydb.RedisDeployment,
	}

	for _, resourceIdent := range resourceIdentsToUpdate {
		if obj, ok := resourceIdent.(providers.ResourceIdentSingle); ok {
			dd := &apps.Deployment{}
			if err := sa.Cache.Get(obj, dd); err != nil {
				if strings.Contains(err.Error(), "not found") {
					continue
				}
			}
			dd.Spec.Template.Spec.ServiceAccountName = sa.Env.GetClowdSAName()
			if err := sa.Cache.Update(obj, dd); err != nil {
				return err
			}
		}
	}

	dList := &apps.DeploymentList{}
	if err := sa.Cache.List(deployment.CoreDeployment, dList); err != nil {
		return err
	}
	for _, d := range dList.Items {
		d.Spec.Template.Spec.ServiceAccountName = app.GetClowdSAName()
		if err := sa.Cache.Update(deployment.CoreDeployment, &d); err != nil {
			return err
		}
	}

	return nil
}
