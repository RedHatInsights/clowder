package providers

import (
	"context"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Labels map[string]string

type Provider struct {
	Client client.Client
	Ctx    context.Context
	Env    *crd.ClowdEnvironment
}

// Configurable is responsible for applying the respective section of
// AppConfig.  It should be called in the app reconciler after all
// provider-specific API calls (e.g. CreateBuckets) have been made.
type Configurable interface {
	Configure(c *config.AppConfig)
}

func StrPtr(s string) *string {
	return &s
}

type makeFn func(o obj.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim)

func MakeComponent(ctx context.Context, cl client.Client, o obj.ClowdObject, suffix string, fn makeFn) error {
	nn := GetNamespacedName(o, suffix)
	dd, svc, pvc := &apps.Deployment{}, &core.Service{}, &core.PersistentVolumeClaim{}
	updates, err := utils.UpdateAllOrErr(ctx, cl, nn, svc, pvc, dd)

	if err != nil {
		return errors.Wrap(fmt.Sprintf("make-%s: get", suffix), err)
	}

	fn(o, dd, svc, pvc)

	if err = utils.ApplyAll(ctx, cl, updates); err != nil {
		return errors.Wrap(fmt.Sprintf("make-%s: upsert", suffix), err)
	}

	return nil
}

func GetNamespacedName(o obj.ClowdObject, suffix string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v", o.GetClowdName(), suffix),
		Namespace: o.GetClowdNamespace(),
	}
}

type ExtractFn func(m map[string][]byte)

func ExtractSecretData(secrets []core.Secret, fn ExtractFn, keys ...string) {
	for _, secret := range secrets {
		allOk := true
		for _, key := range keys {
			if _, ok := secret.Data[key]; !ok {
				allOk = false
				break
			}
		}

		if allOk {
			fn(secret.Data)
		}
	}
}
