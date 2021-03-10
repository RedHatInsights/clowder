package serviceaccount

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func createServiceAccount(cache *providers.ObjectCache, ident providers.ResourceIdent, obj object.ClowdObject, pullSecretNames crd.PullSecrets) error {
	nn := types.NamespacedName{
		Name:      obj.GetClowdSAName(),
		Namespace: obj.GetClowdNamespace(),
	}

	if obj.GetClowdNamespace() == "" {
		err := errors.New("targetNamespace not yet populated")
		err.Requeue = true
		return err
	}

	sa := &core.ServiceAccount{}
	if err := cache.Create(ident, nn, sa); err != nil {
		return err
	}

	sa.ImagePullSecrets = []core.LocalObjectReference{}

	for _, pullSecret := range pullSecretNames {
		sa.ImagePullSecrets = append(sa.ImagePullSecrets, core.LocalObjectReference{
			Name: string(pullSecret),
		})
	}

	labeler := utils.GetCustomLabeler(nil, nn, obj)
	labeler(sa)

	if err := cache.Update(ident, sa); err != nil {
		return err
	}

	return nil
}
