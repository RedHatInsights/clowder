package providers

import (
	"context"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider struct {
	Client client.Client
	Ctx    context.Context
	Env    *crd.ClowdEnvironment
}

// Configurable is responsible for applying the respective section of
// AppConfig.  It should be called in the app reconciler after all
// provider-specific API calls (e.g. CreateBucket) have been made.
type Configurable interface {
	Configure(c *config.AppConfig)
}

// ObjectStoreProvider is the interface for apps to use to configure object
// stores
type ObjectStoreProvider interface {
	Configurable
	CreateBucket(bucket string) error
}

// KafkaProvider is the interface for apps to use to configure kafka topics
type KafkaProvider interface {
	Configurable
	CreateTopic(topic strimzi.KafkaTopicSpec) error
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
	CreateInMemoryDB(app *crd.ClowdApp) error
}

// LoggingProvider is the interface for apps to use to configure logging.  This
// may not be needed on a per-app basis; logging is often only configured on a
// per-environment basis.
type LoggingProvider interface {
	Configurable
	SetupLogging(name string) error
}

func (c *Provider) GetObjectStore() (ObjectStoreProvider, error) {
	objectStoreProvider := c.Env.Spec.ObjectStore.Provider
	switch objectStoreProvider {
	case "minio":
		return NewMinIO(c)
	default:
		return nil, fmt.Errorf("No matching provider for %s", objectStoreProvider)
	}
}

func (c *Provider) GetDatabase() (DatabaseProvider, error) {
	dbProvider := c.Env.Spec.Database.Provider
	switch dbProvider {
	case "local":
		return NewLocalDBProvider(c)
	default:
		return nil, fmt.Errorf("No matching provider for %s", dbProvider)
	}
}

func (c *Provider) GetKafka() (KafkaProvider, error) {
	return nil, nil
}

func (c *Provider) GetInMemoryDB() (InMemoryDBProvider, error) {
	dbProvider := c.Env.Spec.InMemoryDB.Provider
	switch dbProvider {
	case "redis":
		return NewRedis(c)
	default:
		return nil, fmt.Errorf("No matching provider for %s", dbProvider)
	}
}

func (c *Provider) GetLogging() (LoggingProvider, error) {
	return nil, nil
}
