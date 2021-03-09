package controllers

import (
	"context"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createServiceAccount(ctx context.Context, client client.Client, obj object.ClowdObject, pullSecretNames crd.PullSecrets) error {
	nn := types.NamespacedName{
		Name:      obj.GetClowdSAName(),
		Namespace: obj.GetClowdNamespace(),
	}

	sa := &core.ServiceAccount{}

	err := client.Get(ctx, nn, sa)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
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

	if err := update.Apply(ctx, client, sa); err != nil {
		return err
	}

	return nil
}
