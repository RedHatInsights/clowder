package providers

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"
)

type ProviderAccessor struct {
	SetupProvider func(c *Provider) (ClowderProvider, error)
	Order         int
	Name          string
}

type providersRegistration struct {
	Registry []ProviderAccessor
}

func (p *providersRegistration) Len() int {
	return len(p.Registry)
}

func (p *providersRegistration) Swap(i, j int) {
	p.Registry[i], p.Registry[j] = p.Registry[j], p.Registry[i]
}

func (p *providersRegistration) Less(i, j int) bool {
	return p.Registry[i].Order < p.Registry[j].Order
}

func (p *providersRegistration) Register(
	SetupProvider func(c *Provider) (ClowderProvider, error),
	Order int,
	Name string,
) {
	p.Registry = append(p.Registry, ProviderAccessor{
		SetupProvider: SetupProvider,
		Order:         Order,
		Name:          Name,
	})
	sort.Sort(p)
}

var ProvidersRegistration providersRegistration

type Labels map[string]string

type Provider struct {
	Client utils.PClient
	Ctx    context.Context
	Env    *crd.ClowdEnvironment
}

type ClowderProvider interface {
	Provide(app *crd.ClowdApp, c *config.AppConfig) error
}

func StrPtr(s string) *string {
	return &s
}

type makeFn func(o obj.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim, usePVC bool)

func MakeComponent(ctx context.Context, cl utils.PClient, o obj.ClowdObject, suffix string, fn makeFn, usePVC bool) error {
	nn := GetNamespacedName(o, suffix)
	dd, svc, pvc := &apps.Deployment{}, &core.Service{}, &core.PersistentVolumeClaim{}
	ddOld, svcOld, pvcOld := &apps.Deployment{}, &core.Service{}, &core.PersistentVolumeClaim{}

	updates, err := utils.UpdateAllOrErr(ctx, cl, nn, svc, pvc, dd)
	dd.DeepCopyInto(ddOld)
	svc.DeepCopyInto(svcOld)
	pvc.DeepCopyInto(pvcOld)

	if !usePVC {
		delete(updates, pvc)
	}

	if err != nil {
		return errors.Wrap(fmt.Sprintf("make-%s: get", suffix), err)
	}

	fn(o, dd, svc, pvc, usePVC)

	if reflect.DeepEqual(dd.Spec, ddOld.Spec) {
		fmt.Printf("Skipping the update deployment here %v\n", dd.ObjectMeta.Name)
		cl.AddResource(dd)
		delete(updates, dd)
	}

	if reflect.DeepEqual(svc.Spec, svcOld.Spec) {
		fmt.Printf("Skipping the update service here %v\n", svc.ObjectMeta.Name)
		cl.AddResource(svc)
		delete(updates, svc)
	}

	if reflect.DeepEqual(pvc.Spec, pvcOld.Spec) {
		fmt.Printf("Skipping the update persistent here %v\n", pvc.ObjectMeta.Name)
		delete(updates, pvc)
	}

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
