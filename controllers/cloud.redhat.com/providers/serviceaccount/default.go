// Package serviceaccount provides service account and RBAC management for Clowder applications
package serviceaccount

import (
	"fmt"
	"strings"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/database"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/featureflags"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/inmemorydb"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/objectstore"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// CoreDeploymentRoleBinding is the rolebinding for the apps.
var CoreDeploymentRoleBinding = rc.NewMultiResourceIdent(ProvName, "core_deployment_role_binding", &rbac.RoleBinding{})

// CoreDeploymentServiceAccount is the serviceaccount for the apps.
var CoreDeploymentServiceAccount = rc.NewMultiResourceIdent(ProvName, "core_deployment_service_account", &core.ServiceAccount{})

// CoreAppServiceAccount is the serviceaccount for the apps.
var CoreAppServiceAccount = rc.NewSingleResourceIdent(ProvName, "core_app_service_account", &core.ServiceAccount{})

// CoreEnvServiceAccount is the serviceaccount for the env.
var CoreEnvServiceAccount = rc.NewSingleResourceIdent(ProvName, "core_env_service_account", &core.ServiceAccount{})

// IQEServiceAccount is the serviceaccount for the iqe testing.
var IQEServiceAccount = rc.NewMultiResourceIdent(ProvName, "iqe_service_account", &core.ServiceAccount{})

// IQERoleBinding is the reolbinding for the env.
var IQERoleBinding = rc.NewMultiResourceIdent(ProvName, "iqe_role_binding", &rbac.RoleBinding{})

type serviceaccountProvider struct {
	providers.Provider
}

// NewServiceAccountProvider creates a new service account provider instance
func NewServiceAccountProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CoreDeploymentRoleBinding,
		CoreDeploymentServiceAccount,
		CoreAppServiceAccount,
		CoreEnvServiceAccount,
		IQEServiceAccount,
		IQERoleBinding,
	)
	return &serviceaccountProvider{Provider: *p}, nil
}

func (sa *serviceaccountProvider) EnvProvide() error {
	if err := createServiceAccountForClowdObj(sa.Cache, CoreEnvServiceAccount, sa.Env); err != nil {
		return err
	}

	resourceIdentsToUpdate := []rc.ResourceIdent{
		featureflags.LocalFFDBDeployment,
		featureflags.LocalFFDeployment,
		objectstore.MinioDeployment,
		database.SharedDBDeployment,
	}

	for _, resourceIdent := range resourceIdentsToUpdate {
		if obj, ok := resourceIdent.(rc.ResourceIdentSingle); ok {
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
	return nil
}

func (sa *serviceaccountProvider) Provide(app *crd.ClowdApp) error {

	if err := createIQEServiceAccounts(&sa.Provider, app); err != nil {
		return err
	}

	if err := createServiceAccountForClowdObj(sa.Cache, CoreAppServiceAccount, app); err != nil {
		return err
	}

	resourceIdentsToUpdate := []rc.ResourceIdent{
		database.LocalDBDeployment,
		inmemorydb.RedisDeployment,
	}

	for _, resourceIdent := range resourceIdentsToUpdate {
		if obj, ok := resourceIdent.(rc.ResourceIdentSingle); ok {
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
		innerDeployment := dep
		nn := app.GetDeploymentNamespacedName(&innerDeployment)

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

		if err := CreateRoleBinding(sa.Cache, CoreDeploymentRoleBinding, nn, labeler, innerDeployment.K8sAccessLevel); err != nil {
			return err
		}

	}

	return nil
}

func createIQEServiceAccounts(p *providers.Provider, app *crd.ClowdApp) error {

	accessLevel := p.Env.Spec.Providers.Testing.K8SAccessLevel

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("iqe-%s", p.Env.Name),
		Namespace: app.Namespace,
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

	return nil
}
