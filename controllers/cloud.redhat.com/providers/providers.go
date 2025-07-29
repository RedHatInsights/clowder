package providers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"

	"github.com/go-logr/logr"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
)

// MinioSecret is the resource ident for the Minio secret object.
var RootSecret = rc.NewSingleResourceIdent("root", "root_env", &core.Secret{})

type providerAccessor struct {
	FinalizeProvider func(c *Provider) error
	SetupProvider    func(c *Provider) (ClowderProvider, error)
	Order            int
	Name             string
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
	setupProvider func(c *Provider) (ClowderProvider, error),
	order int,
	name string,
	finalizeProvider ...func(c *Provider) error,
) {
	acc := providerAccessor{
		SetupProvider: setupProvider,
		Order:         order,
		Name:          name,
	}
	if len(finalizeProvider) == 1 {
		acc.FinalizeProvider = finalizeProvider[0]
	}
	p.Registry = append(p.Registry, acc)
	sort.Sort(p)
}

// ProvidersRegistration is an instance of the provider registration system. It is responsible for
// adding new providers to the registry so that they can be executed in the correct order.
var ProvidersRegistration providersRegistration

// Labels is a map containing a key/value style map intended to hold k8s label information.
type Labels map[string]string

// Provider is a struct that holds a client/context and ClowdEnvironment object.
type Provider struct {
	Client    client.Client
	Ctx       context.Context
	Env       *crd.ClowdEnvironment
	Cache     *rc.ObjectCache
	Log       logr.Logger
	Config    *config.AppConfig
	HashCache *hashcache.HashCache
}

// GetClient returns the Kubernetes client
func (prov *Provider) GetClient() client.Client {
	return prov.Client
}

// GetCtx returns the context for provider operations
func (prov *Provider) GetCtx() context.Context {
	return prov.Ctx
}

// GetEnv returns the ClowdEnvironment associated with this provider
func (prov *Provider) GetEnv() *crd.ClowdEnvironment {
	return prov.Env
}

// GetCache returns the object cache used by the provider
func (prov *Provider) GetCache() *rc.ObjectCache {
	return prov.Cache
}

// GetLog returns the logger instance for the provider
func (prov *Provider) GetLog() logr.Logger {
	return prov.Log
}

// GetConfig returns the application configuration
func (prov *Provider) GetConfig() *config.AppConfig {
	return prov.Config
}

type RootProvider interface {
	GetClient() client.Client
	GetCtx() context.Context
	GetEnv() *crd.ClowdEnvironment
	GetCache() *rc.ObjectCache
	GetLog() logr.Logger
	GetConfig() *config.AppConfig
}

// ClowderProvider is an interface providing a way for a provider to perform its duty.
type ClowderProvider interface {
	// Provide is the main function that performs the duty of the provider on a ClowdApp object, as
	// opposed to a ClowdEnvironment object.
	Provide(app *crd.ClowdApp) error
	EnvProvide() error
	GetConfig() *config.AppConfig
}

type makeFnCache func(env *crd.ClowdEnvironment, o obj.ClowdObject, objMap ObjectMap, usePVC bool, nodePort bool) error

func createResource(cache *rc.ObjectCache, resourceIdent rc.ResourceIdent, nn types.NamespacedName) (client.Object, error) {
	gvks, nok, err := cache.GetScheme().ObjectKinds(resourceIdent.GetType())

	if err != nil {
		return nil, err
	}

	if nok {
		return nil, fmt.Errorf("this type is unknown")
	}

	gvk := gvks[0]

	cobj, err := cache.GetScheme().New(gvk)
	nobj := cobj.(client.Object)

	if err != nil {
		return nil, err
	}

	err = cache.Create(resourceIdent, nn, nobj)

	if err != nil {
		return nil, fmt.Errorf("error creating object: %s%s %w", nn.Name, nn.Namespace, err)
	}
	return nobj, nil
}

func updateResource(cache *rc.ObjectCache, resourceIdent rc.ResourceIdent, object client.Object) error {
	err := cache.Update(resourceIdent, object)

	gvks, nok, werr := cache.GetScheme().ObjectKinds(resourceIdent.GetType())

	if werr != nil {
		return werr
	}

	if nok {
		return fmt.Errorf("this type is unknown")
	}

	gvk := gvks[0]

	if err != nil {
		return fmt.Errorf("error updating object: %s %s %s %w", object.GetName(), object.GetNamespace(), gvk.Kind, err)
	}
	return nil
}

// ObjectMap providers a map of ResourceIdents to objects, it is used internally and for testing.
type ObjectMap map[rc.ResourceIdent]client.Object

// CachedMakeComponent is a generalised function that, given a ClowdObject will make the given service,
// deployment and PVC, based on the makeFn that is passed in.
func CachedMakeComponent(prov RootProvider, objList []rc.ResourceIdent, o obj.ClowdObject, suffix string, fn makeFnCache, usePVC bool) error {
	nn := GetNamespacedName(o, suffix)

	makeFnMap := make(map[rc.ResourceIdent]client.Object)

	for _, v := range objList {
		obj, err := createResource(prov.GetCache(), v, nn)

		if err != nil {
			return errors.Wrap(fmt.Sprintf("make-%s: get", suffix), err)
		}

		makeFnMap[v] = obj

	}

	err := fn(prov.GetEnv(), o, makeFnMap, usePVC, prov.GetEnv().IsNodePort())
	if err != nil {
		return fmt.Errorf("could not make component: %w", err)
	}

	for k, v := range makeFnMap {
		err := updateResource(prov.GetCache(), k, v)

		if err != nil {
			return errors.Wrap(fmt.Sprintf("make-%s: get", suffix), err)
		}
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

// ExtractFn is a function that can extract secret data from a function, the result of this function
// is usually declared as part of the function so no arguments are passed.
type ExtractFn func(m *core.Secret)

// ExtractFnAnno is just like ExtractFn except it reads in the value of an annotation
type ExtractFnAnno func(m *core.Secret, annoVal string)

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
			sec := secret
			fn(&sec)
		}
	}
}

// ExtractSecretDataAnno is just like ExtractSecretData except it expects an
// annotation to be present on the secret that will provide one of the values
// needed to extract the data out of the secret
func ExtractSecretDataAnno(secrets []core.Secret, fn ExtractFnAnno, annoKey string, keys ...string) {
	for _, secret := range secrets {
		allOk := true

		if _, ok := secret.Annotations[annoKey]; !ok {
			allOk = false
		}

		for _, key := range keys {
			if _, ok := secret.Data[key]; !ok {
				allOk = false
				break
			}
		}

		if allOk {
			for _, value := range strings.Split(secret.Annotations[annoKey], ",") {
				sec := secret
				fn(&sec, value)
			}
		}
	}
}

// MakeOrGetSecret tries to get the secret described by nn, if it exists it populates a map with the
// key/value pairs from the secret. If it doesn't exist the dataInit function is run and the
// resulting data is returned, as well as the secret being created.
func MakeOrGetSecret(obj obj.ClowdObject, cache *rc.ObjectCache, resourceIdent rc.ResourceIdent, nn types.NamespacedName, dataInit func() map[string]string) (*map[string]string, error) {
	secret := &core.Secret{}
	if err := cache.Create(resourceIdent, nn, secret); err != nil {
		return nil, err
	}

	data := make(map[string]string)

	if len(secret.Data) == 0 {
		data = dataInit()
		secret.StringData = data

		secret.Name = nn.Name
		secret.Namespace = nn.Namespace
		secret.OwnerReferences = []metav1.OwnerReference{obj.MakeOwnerReference()}
		secret.Type = core.SecretTypeOpaque

	} else {
		for k, v := range secret.Data {
			(data)[k] = string(v)
		}
	}

	if err := cache.Update(resourceIdent, secret); err != nil {
		return nil, err
	}

	return &data, nil
}
