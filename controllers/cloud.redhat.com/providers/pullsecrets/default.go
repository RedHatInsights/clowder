// Package pullsecrets provides pull secret management functionality for Clowder applications
package pullsecrets

import (
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/serviceaccount"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// CoreEnvPullSecrets is the pull_secrets for the app.
var CoreEnvPullSecrets = rc.NewMultiResourceIdent(ProvName, "core_env_pull_secrets", &core.Secret{})

type pullsecretProvider struct {
	providers.Provider
}

// NewPullSecretProvider returns a new End provider run at the end of the provider set.
func NewPullSecretProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CoreEnvPullSecrets,
	)
	return &pullsecretProvider{Provider: *p}, nil
}

func (ps *pullsecretProvider) EnvProvide() error {
	secList, err := CopyPullSecrets(&ps.Provider, ps.Env.Status.TargetNamespace, ps.Env)

	if err != nil {
		return err
	}

	sa := &core.ServiceAccount{}
	if err := ps.Cache.Get(serviceaccount.CoreEnvServiceAccount, sa); err != nil {
		return err
	}

	addAllSecrets(secList, sa)

	return ps.Cache.Update(serviceaccount.CoreEnvServiceAccount, sa)
}

func (ps *pullsecretProvider) Provide(app *crd.ClowdApp) error {

	secList, err := CopyPullSecrets(&ps.Provider, app.Namespace, app)

	if err != nil {
		return err
	}

	iqeServiceAccounts := &core.ServiceAccountList{}
	if err := ps.Cache.List(serviceaccount.IQEServiceAccount, iqeServiceAccounts); err != nil {
		return err
	}

	for _, iqeSA := range iqeServiceAccounts.Items {
		innerIQESA := iqeSA
		addAllSecrets(secList, &innerIQESA)

		if err := ps.Cache.Update(serviceaccount.IQEServiceAccount, &innerIQESA); err != nil {
			return err
		}
	}

	saList := &core.ServiceAccountList{}
	if err := ps.Cache.List(serviceaccount.CoreDeploymentServiceAccount, saList); err != nil {
		return err
	}

	for _, sa := range saList.Items {
		innerSA := sa
		addAllSecrets(secList, &innerSA)

		if err := ps.Cache.Update(serviceaccount.CoreDeploymentServiceAccount, &innerSA); err != nil {
			return err
		}
	}

	sa := &core.ServiceAccount{}
	if err := ps.Cache.Get(serviceaccount.CoreAppServiceAccount, sa); err != nil {
		return err
	}

	addAllSecrets(secList, sa)

	return ps.Cache.Update(serviceaccount.CoreAppServiceAccount, sa)
}

func CopyPullSecrets(prov *providers.Provider, namespace string, obj object.ClowdObject) ([]string, error) {

	var secList []string

	for _, pullSecretName := range prov.Env.Spec.Providers.PullSecrets {

		sourcePullSecObj := &core.Secret{}
		if err := prov.Client.Get(prov.Ctx, types.NamespacedName{
			Name:      pullSecretName.Name,
			Namespace: pullSecretName.Namespace,
		}, sourcePullSecObj); err != nil {
			return nil, err
		}

		_, err := prov.HashCache.CreateOrUpdateObject(sourcePullSecObj, true)
		if err != nil {
			return nil, err
		}

		if err = prov.HashCache.AddClowdObjectToObject(obj, sourcePullSecObj); err != nil {
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

		// check if the secret already exists, if not create it
		if err := prov.Cache.Get(CoreEnvPullSecrets, newPullSecObj, newSecNN); err != nil {
			if err := prov.Cache.Create(CoreEnvPullSecrets, newSecNN, newPullSecObj); err != nil {
				return nil, err
			}
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

func addAllSecrets(secList []string, sa *core.ServiceAccount) {

	newSecrets := []core.LocalObjectReference{}

	for _, existingSec := range sa.ImagePullSecrets {
		if strings.HasPrefix(existingSec.Name, fmt.Sprintf("%s-dockercfg", sa.Name)) {
			newSecrets = append(newSecrets, existingSec)
		}
	}

	for _, pullSecretName := range secList {

		newSecrets = append(newSecrets, core.LocalObjectReference{
			Name: pullSecretName,
		})
	}

	sa.ImagePullSecrets = newSecrets
}
