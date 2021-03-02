package providers

import (
	"context"
	"fmt"
	"sort"

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

type providerAccessor struct {
	SetupProvider func(c *Provider) (ClowderProvider, error)
	Order         int
	Name          string
}

type providersRegistration struct {
	Registry []providerAccessor
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
	p.Registry = append(p.Registry, providerAccessor{
		SetupProvider: SetupProvider,
		Order:         Order,
		Name:          Name,
	})
	sort.Sort(p)
}

// ProvidersRegistration is an instance of the provider registration system. It is responsible for
// adding new providers to the registry so that they can be executed in the correct order.
var ProvidersRegistration providersRegistration

// Labels is a map containing a key/value style map intended to hold k8s label information.
type Labels map[string]string

// Provider is a struct that holds a client/context and ClowdEnvironment object.
type Provider struct {
	Client client.Client
	Ctx    context.Context
	Env    *crd.ClowdEnvironment
}

// ClowderProvider is an interface providing a way for a provider to perform its duty.
type ClowderProvider interface {
	// Provide is the main function that performs the duty of the provider on a ClowdApp object, as opposed
	// to a ClowdEnvironment object.
	Provide(app *crd.ClowdApp, c *config.AppConfig) error
}

// StrPtr returns a pointer to a string.
func StrPtr(s string) *string {
	return &s
}

type makeFn func(o obj.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim, usePVC bool)

// MakeComponent is a generalised function that, given a ClowdObject will make the given service,
// deployment and PVC, based on the makeFn that is passed in.
func MakeComponent(ctx context.Context, cl client.Client, o obj.ClowdObject, suffix string, fn makeFn, usePVC bool) error {
	nn := GetNamespacedName(o, suffix)
	dd, svc, pvc := &apps.Deployment{}, &core.Service{}, &core.PersistentVolumeClaim{}
	updates, err := utils.UpdateAllOrErr(ctx, cl, nn, svc, pvc, dd)

	if !usePVC {
		delete(updates, pvc)
	}

	if err != nil {
		return errors.Wrap(fmt.Sprintf("make-%s: get", suffix), err)
	}

	fn(o, dd, svc, pvc, usePVC)

	if err = utils.ApplyAll(ctx, cl, updates); err != nil {
		return errors.Wrap(fmt.Sprintf("make-%s: upsert", suffix), err)
	}

	return nil
}

// GetNamespacedName returns a unique name of an object in the format name-suffix.
func GetNamespacedName(o obj.ClowdObject, suffix string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v", o.GetClowdName(), suffix),
		Namespace: o.GetClowdNamespace(),
	}
}

// ExtractFn is a function that can extract secret data from a function, the result
// of this function is usually declared as part of the function so no arguments are
// passed.
type ExtractFn func(m map[string][]byte)

// ExtractSecretData takes a list of secrets, checks that the correct 'keys' are present and then
// runs the extract function on them.
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
