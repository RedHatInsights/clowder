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
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	rbac "k8s.io/api/rbac/v1"
)

type serviceaccountProvider struct {
	providers.Provider
}

func NewServiceAccountProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	if err := createServiceAccountForClowdObj(p.Cache, CoreEnvServiceAccount, p.Env, p.Env.Spec.Providers.PullSecrets); err != nil {
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

	if err := createServiceAccountForClowdObj(sa.Cache, CoreAppServiceAccount, app, sa.Env.Spec.Providers.PullSecrets); err != nil {
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

	for _, dep := range app.Spec.Deployments {
		d := &apps.Deployment{}
		nn := app.GetDeploymentNamespacedName(&dep)

		if err := sa.Cache.Get(deployment.CoreDeployment, d, nn); err != nil {
			return err
		}

		labeler := utils.GetCustomLabeler(nil, nn, app)

		if err := CreateServiceAccount(sa.Cache, CoreDeploymentServiceAccount, sa.Env.Spec.Providers.PullSecrets, nn, labeler); err != nil {
			return err
		}

		d.Spec.Template.Spec.ServiceAccountName = nn.Name
		if err := sa.Cache.Update(deployment.CoreDeployment, d); err != nil {
			return err
		}

		if dep.K8sAccessLevel == "default" || dep.K8sAccessLevel == "" {
			continue
		}

		rb := &rbac.RoleBinding{}

		if err := sa.Cache.Create(CoreDeploymentRoleBinding, nn, rb); err != nil {
			return err
		}

		labeler(rb)

		rb.Subjects = []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      nn.Name,
			Namespace: nn.Namespace,
		}}
		rb.RoleRef = rbac.RoleRef{
			Kind: "ClusterRole",
		}

		switch dep.K8sAccessLevel {
		case "view":
			rb.RoleRef.Name = "view"
		case "edit":
			rb.RoleRef.Name = "edit"
		}

		if err := sa.Cache.Update(CoreDeploymentRoleBinding, rb); err != nil {
			return err
		}
	}

	return nil
}
