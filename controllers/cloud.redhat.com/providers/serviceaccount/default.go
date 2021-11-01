package serviceaccount

import (
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/database"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/featureflags"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/inmemorydb"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/kafka"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/objectstore"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

type serviceaccountProvider struct {
	providers.Provider
}

func NewServiceAccountProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	if err := createIQEServiceAccounts(p, p.Env); err != nil {
		return nil, err
	}

	if err := createServiceAccountForClowdObj(p.Cache, CoreEnvServiceAccount, p.Env); err != nil {
		return nil, err
	}

	resourceIdentsToUpdate := []providers.ResourceIdent{
		featureflags.LocalFFDBDeployment,
		kafka.LocalKafkaDeployment,
		kafka.LocalZookeeperDeployment,
		objectstore.MinioDeployment,
		database.LocalDBDeployment,
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

	if err := createServiceAccountForClowdObj(sa.Cache, CoreAppServiceAccount, app); err != nil {
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
			dd.Spec.Template.Spec.ServiceAccountName = app.GetClowdSAName()
			if err := sa.Cache.Update(obj, dd); err != nil {
				return err
			}
		}
	}

	for _, dep := range app.Spec.Deployments {
		d := &apps.Deployment{}
		nn := app.GetDeploymentNamespacedName(&dep)

		if err := sa.Cache.Get(deployment.CoreDeployment, d, nn); err != nil {
			return err
		}

		labeler := utils.GetCustomLabeler(nil, nn, app)

		if err := CreateServiceAccount(sa.Cache, CoreDeploymentServiceAccount, nn, labeler); err != nil {
			return err
		}

		d.Spec.Template.Spec.ServiceAccountName = nn.Name
		if err := sa.Cache.Update(deployment.CoreDeployment, d); err != nil {
			return err
		}

		if err := CreateRoleBinding(sa.Cache, CoreDeploymentRoleBinding, nn, labeler, dep.K8sAccessLevel); err != nil {
			return err
		}

	}

	return nil
}

func createIQEServiceAccounts(p *providers.Provider, env *crd.ClowdEnvironment) error {

	accessLevel := env.Spec.Providers.Testing.K8SAccessLevel

	namespaces, err := env.GetNamespacesInEnv(p.Ctx, p.Client)

	if err != nil {
		return err
	}

	for _, namespace := range namespaces {

		nn := types.NamespacedName{
			Name:      fmt.Sprintf("iqe-%s", env.Name),
			Namespace: namespace,
		}

		labeler := utils.GetCustomLabeler(nil, nn, p.Env)
		if err := CreateServiceAccount(p.Cache, IQEServiceAccount, nn, labeler); err != nil {
			return err
		}

		switch accessLevel {
		// Use edit level service account to create and delete resources
		// one per app when the app is created
		case "edit", "view":
			if err := CreateRoleBinding(p.Cache, IQERoleBinding, nn, labeler, accessLevel); err != nil {
				return err
			}

		default:
		}
	}
	return nil
}
