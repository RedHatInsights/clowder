package providers

import (
	"context"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type labels map[string]string

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

// ObjectStoreProvider is the interface for apps to use to configure object
// stores
type ObjectStoreProvider interface {
	Configurable
	CreateBuckets(app *crd.ClowdApp) error
}

// KafkaProvider is the interface for apps to use to configure kafka topics
type KafkaProvider interface {
	Configurable
	CreateTopic(nn types.NamespacedName, topic *strimzi.KafkaTopicSpec) error
}

// DatabaseProvider is the interface for apps to use to configure databases
type DatabaseProvider interface {
	Configurable
	CreateDatabase(app *crd.ClowdApp) error
}

// InMemoryDBProvider is the interface for apps to use to configure in-memory
// databases
type InMemoryDBProvider interface {
	Configurable
}

// LoggingProvider is the interface for apps to use to configure logging.  This
// may not be needed on a per-app basis; logging is often only configured on a
// per-environment basis.
type LoggingProvider interface {
	Configurable
	SetUpLogging(app *crd.ClowdApp) error
}

func (c *Provider) GetObjectStore() (ObjectStoreProvider, error) {
	objectStoreProvider := c.Env.Spec.ObjectStore.Provider
	switch objectStoreProvider {
	case "minio":
		return NewMinIO(c)
	case "app-interface":
		return &AppInterfaceObjectstoreProvider{Provider: *c}, nil
	default:
		errStr := fmt.Sprintf("No matching object store provider for %s", objectStoreProvider)
		return nil, errors.New(errStr)
	}
}

func (c *Provider) GetDatabase() (DatabaseProvider, error) {
	dbProvider := c.Env.Spec.Database.Provider
	switch dbProvider {
	case "local":
		return NewLocalDBProvider(c)
	default:
		errStr := fmt.Sprintf("No matching db provider for %s", dbProvider)
		return nil, errors.New(errStr)
	}
}

func (c *Provider) GetKafka() (KafkaProvider, error) {
	kafkaProvider := c.Env.Spec.Kafka.Provider
	switch kafkaProvider {
	case "operator":
		return NewStrimzi(c)
	case "local":
		return NewLocalKafka(c)
	default:
		errStr := fmt.Sprintf("No matching kafka provider for %s", kafkaProvider)
		return nil, errors.New(errStr)
	}
}

func (c *Provider) GetInMemoryDB() (InMemoryDBProvider, error) {
	inMemoryDBProvider := c.Env.Spec.InMemoryDB.Provider
	switch inMemoryDBProvider {
	case "redis":
		return NewLocalRedis(c)
	default:
		errStr := fmt.Sprintf("No matching in-memory db provider for %s", inMemoryDBProvider)
		return nil, errors.New(errStr)
	}
}

func (c *Provider) GetLogging() (LoggingProvider, error) {
	logProvider := c.Env.Spec.Logging.Provider
	switch logProvider {
	case "app-interface":
		return NewAppInterfaceLogging(c)
	case "none":
		return nil, nil
	default:
		errStr := fmt.Sprintf("No matching logging provider for %s", logProvider)
		return nil, errors.New(errStr)
	}
}

func (c *Provider) SetUpEnvironment() error {
	var err error

	if _, err = c.GetObjectStore(); err != nil {
		return errors.Wrap("setupenv: getobjectstore", err)
	}

	if _, err = c.GetDatabase(); err != nil {
		return errors.Wrap("setupenv: getdatabase", err)
	}

	if _, err = c.GetKafka(); err != nil {
		return errors.Wrap("setupenv: getkafka", err)
	}

	if _, err = c.GetInMemoryDB(); err != nil {
		return errors.Wrap("setupenv: getinmemorydb", err)
	}

	if _, err = c.GetLogging(); err != nil {
		return errors.Wrap("setupenv: getlogging", err)
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}

type makeFn func(env *crd.ClowdEnvironment, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim)

func makeComponent(p *Provider, suffix string, fn makeFn) error {
	nn := getNamespacedName(p.Env, suffix)
	dd, svc, pvc := &apps.Deployment{}, &core.Service{}, &core.PersistentVolumeClaim{}
	updates, err := utils.UpdateAllOrErr(p.Ctx, p.Client, nn, svc, pvc, dd)

	if err != nil {
		return errors.Wrap(fmt.Sprintf("make-%s: get", suffix), err)
	}

	fn(p.Env, dd, svc, pvc)

	if err = utils.ApplyAll(p.Ctx, p.Client, updates); err != nil {
		return errors.Wrap(fmt.Sprintf("make-%s: upsert", suffix), err)
	}

	return nil
}

func getNamespacedName(env *crd.ClowdEnvironment, suffix string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v", env.Name, suffix),
		Namespace: env.Spec.Namespace,
	}
}
