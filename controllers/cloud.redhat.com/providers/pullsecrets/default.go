package pullsecrets

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/serviceaccount"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type pullsecretProvider struct {
	providers.Provider
}

// CoreConfigSecret is the config that is presented as the cdappconfig.json file.
var CoreConfigSecret = providers.NewSingleResourceIdent(ProvName, "core_config_secret", &core.Secret{})

// NewPullSecretProvider returns a new End provider run at the end of the provider set.
func NewPullSecretProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	var appList *crd.ClowdAppList
	var err error

	if appList, err = p.Env.GetAppsInEnv(p.Ctx, p.Client); err != nil {
		return nil, err
	}

	namespaceSet := map[string]bool{}

	for _, app := range appList.Items {
		namespaceSet[app.Namespace] = true
	}

	secList, err := copyPullSecrets(p, namespaceSet)

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

	iqeServiceAccounts := &core.ServiceAccountList{}
	if err := p.Cache.List(serviceaccount.IQEServiceAccount, iqeServiceAccounts); err != nil {
		return nil, err
	}

	for _, iqeSA := range iqeServiceAccounts.Items {
		addAllSecrets(secList, &iqeSA)

		if err := p.Cache.Update(serviceaccount.IQEServiceAccount, &iqeSA); err != nil {
			return nil, err
		}
	}

	return &pullsecretProvider{Provider: *p}, nil
}

func copyPullSecrets(prov *providers.Provider, namespaceList map[string]bool) ([]string, error) {
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

		for namespace := range namespaceList {

			newPullSecObj := &core.Secret{}

			newSecNN := types.NamespacedName{
				Name:      secName,
				Namespace: namespace,
			}

			labeler := utils.GetCustomLabeler(map[string]string{}, newSecNN, prov.Env)

			if err := prov.Cache.Create(CoreEnvPullSecrets, newSecNN, newPullSecObj); err != nil {
				return nil, err
			}

			newPullSecObj.Data = sourcePullSecObj.Data
			newPullSecObj.Type = sourcePullSecObj.Type

			labeler(newPullSecObj)

			newPullSecObj.Name = newSecNN.Name
			newPullSecObj.Namespace = newSecNN.Namespace

			if err := prov.Cache.Update(CoreEnvPullSecrets, newPullSecObj); err != nil {
				return nil, err
			}
		}
	}
	return secList, nil
}

func (ps *pullsecretProvider) getSecretList(app *crd.ClowdApp) []string {
	secList := []string{}
	for _, secret := range ps.Env.Spec.Providers.PullSecrets {
		if secret.Namespace == app.Namespace {
			secList = append(secList, secret.Name)
		} else {
			secList = append(secList, fmt.Sprintf("%s-clowder-copy", secret.Name))
		}
	}
	return secList
}

func (ps *pullsecretProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	if err := ps.annotateServiceAccounts(ps.getSecretList(app)); err != nil {
		return err
	}

	return nil
}

func (ps *pullsecretProvider) annotateServiceAccounts(secList []string) error {
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
