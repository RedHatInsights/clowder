package pullsecrets

import (
	"context"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/serviceaccount"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type pullsecretProvider struct {
	providers.Provider
}

// CoreConfigSecret is the config that is presented as the cdappconfig.json file.
var CoreConfigSecret = providers.NewSingleResourceIdent(ProvName, "core_config_secret", &core.Secret{})

// NewPullSecretProvider returns a new End provider run at the end of the provider set.
func NewPullSecretProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	secList, err := getSecList(p.Ctx, p.Client, *p.Cache, *p.Env, p.Env.Status.TargetNamespace, p.Env, true)

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

func (ps *pullsecretProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	var secList []string
	var err error

	if app.Namespace != ps.Env.Status.TargetNamespace {
		secList, err = getSecList(ps.Ctx, ps.Client, *ps.Cache, *ps.Env, app.Namespace, app, true)
	} else {
		secList, err = getSecList(ps.Ctx, ps.Client, *ps.Cache, *ps.Env, app.Namespace, app, false)
	}
	if err != nil {
		return err
	}

	if err := ps.annotateServiceAccounts(secList); err != nil {
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

func getSecList(ctx context.Context, client client.Client, cache providers.ObjectCache, env crd.ClowdEnvironment, targetNamespace string, owner object.ClowdObject, createSecret bool) ([]string, error) {
	secList := []string{}

	for _, pullSecretName := range env.Spec.Providers.PullSecrets {
		if pullSecretName.Namespace == targetNamespace {
			secList = append(secList, pullSecretName.Name)
			continue
		}

		sourcePullSecObj := &core.Secret{}
		if err := client.Get(ctx, types.NamespacedName{
			Name:      pullSecretName.Name,
			Namespace: pullSecretName.Namespace,
		}, sourcePullSecObj); err != nil {
			return nil, err
		}

		newPullSecObj := &core.Secret{}

		secName := fmt.Sprintf("%s-clowder-copy", pullSecretName.Name)

		newSecNN := types.NamespacedName{
			Name:      secName,
			Namespace: targetNamespace,
		}

		secList = append(secList, secName)

		if !createSecret {
			continue
		}

		labeler := utils.GetCustomLabeler(map[string]string{}, newSecNN, owner)

		if err := cache.Create(CoreEnvPullSecrets, newSecNN, newPullSecObj); err != nil {
			return nil, err
		}

		newPullSecObj.Data = sourcePullSecObj.Data
		newPullSecObj.Type = sourcePullSecObj.Type

		labeler(newPullSecObj)

		newPullSecObj.Name = newSecNN.Name
		newPullSecObj.Namespace = newSecNN.Namespace

		if err := cache.Update(CoreEnvPullSecrets, newPullSecObj); err != nil {
			return nil, err
		}
	}
	return secList, nil
}
