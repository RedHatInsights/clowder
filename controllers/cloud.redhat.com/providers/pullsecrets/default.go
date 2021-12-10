package pullsecrets

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/serviceaccount"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

type pullsecretProvider struct {
	providers.Provider
}

// CoreConfigSecret is the config that is presented as the cdappconfig.json file.
var CoreConfigSecret = rc.NewSingleResourceIdent(ProvName, "core_config_secret", &core.Secret{})

// NewPullSecretProvider returns a new End provider run at the end of the provider set.
func NewPullSecretProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	secList, err := copyPullSecrets(p, p.Env.Status.TargetNamespace, p.Env)

	if err != nil {
		return nil, err
	}

	sa := &core.ServiceAccount{}
	if err := p.Cache.Get(serviceaccount.CoreEnvServiceAccount, sa); err != nil {
		return nil, err
	}

	addAllSecrets(secList, sa)

	if err := p.Cache.Update(serviceaccount.CoreEnvServiceAccount, sa); err != nil {
		return nil, err
	}

	return &pullsecretProvider{Provider: *p}, nil
}

func copyPullSecrets(prov *providers.Provider, namespace string, obj object.ClowdObject) ([]string, error) {

	var secList []string

	for _, pullSecretName := range prov.Env.Spec.Providers.PullSecrets {

		sourcePullSecObj := &core.Secret{}
		if err := prov.Client.Get(prov.Ctx, types.NamespacedName{
			Name:      pullSecretName.Name,
			Namespace: pullSecretName.Namespace,
		}, sourcePullSecObj); err != nil {
			return nil, err
		}

		secName := fmt.Sprintf("%s-%s-clowder-copy", prov.Env.Name, pullSecretName.Name)
		secList = append(secList, secName)

		newPullSecObj := &core.Secret{}

		newSecNN := types.NamespacedName{
			Name:      secName,
			Namespace: namespace,
		}

		if obj.GroupVersionKind().Kind == "ClowdApp" && obj.GetClowdNamespace() == prov.Env.Spec.TargetNamespace {
			continue
		}

		if err := prov.Cache.Create(CoreEnvPullSecrets, newSecNN, newPullSecObj); err != nil {
			return nil, err
		}

		newPullSecObj.Data = sourcePullSecObj.Data
		newPullSecObj.Type = sourcePullSecObj.Type

		labeler := utils.GetCustomLabeler(map[string]string{}, newSecNN, prov.Env)
		labeler(newPullSecObj)

		newPullSecObj.Name = newSecNN.Name
		newPullSecObj.Namespace = newSecNN.Namespace

		if err := prov.Cache.Update(CoreEnvPullSecrets, newPullSecObj); err != nil {
			return nil, err
		}
	}
	return secList, nil
}

func (ps *pullsecretProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	secList, err := copyPullSecrets(&ps.Provider, app.Namespace, app)

	if err != nil {
		return err
	}

	iqeServiceAccounts := &core.ServiceAccountList{}
	if err := ps.Cache.List(serviceaccount.IQEServiceAccount, iqeServiceAccounts); err != nil {
		return err
	}

	for _, iqeSA := range iqeServiceAccounts.Items {
		addAllSecrets(secList, &iqeSA)

		if err := ps.Cache.Update(serviceaccount.IQEServiceAccount, &iqeSA); err != nil {
			return err
		}
	}

	saList := &core.ServiceAccountList{}
	if err := ps.Cache.List(serviceaccount.CoreDeploymentServiceAccount, saList); err != nil {
		return err
	}

	for _, sa := range saList.Items {

		addAllSecrets(secList, &sa)

		if err := ps.Cache.Update(serviceaccount.CoreDeploymentServiceAccount, &sa); err != nil {
			return err
		}
	}

	sa := &core.ServiceAccount{}
	if err := ps.Cache.Get(serviceaccount.CoreAppServiceAccount, sa); err != nil {
		return err
	}

	addAllSecrets(secList, sa)

	if err := ps.Cache.Update(serviceaccount.CoreAppServiceAccount, sa); err != nil {
		return err
	}

	return nil
}

func addAllSecrets(secList []string, sa *core.ServiceAccount) {
	sa.ImagePullSecrets = []core.LocalObjectReference{}

	for _, pullSecretName := range secList {

		sa.ImagePullSecrets = append(sa.ImagePullSecrets, core.LocalObjectReference{
			Name: pullSecretName,
		})
	}
}
