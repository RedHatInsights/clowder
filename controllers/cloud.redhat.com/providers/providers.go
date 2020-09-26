package providers

import (
	"context"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"k8s.io/apimachinery/pkg/types"
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
	CreateInMemoryDB(app *crd.ClowdApp) error
}

// LoggingProvider is the interface for apps to use to configure logging.  This
// may not be needed on a per-app basis; logging is often only configured on a
// per-environment basis.
type LoggingProvider interface {
	Configurable
	SetUpLogging(nn types.NamespacedName) error
}

func (c *Provider) GetObjectStore() (ObjectStoreProvider, error) {
	objectStoreProvider := c.Env.Spec.ObjectStore.Provider
	switch objectStoreProvider {
	case "minio":
		return NewMinIO(c)
	case "app-interface":
		return &AppInterfaceProvider{Provider: *c}, nil
	default:
		return nil, fmt.Errorf("No matching object store provider for %s", objectStoreProvider)
	}
}

func (c *Provider) GetDatabase() (DatabaseProvider, error) {
	dbProvider := c.Env.Spec.Database.Provider
	switch dbProvider {
	case "local":
		return NewLocalDBProvider(c)
	default:
		return nil, fmt.Errorf("No matching db provider for %s", dbProvider)
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
		return nil, fmt.Errorf("No matching kafka provider for %s", kafkaProvider)
	}
}

func (c *Provider) GetInMemoryDB() (InMemoryDBProvider, error) {
	dbProvider := c.Env.Spec.InMemoryDB.Provider
	switch dbProvider {
	case "redis":
		return NewRedis(c)
	default:
		return nil, fmt.Errorf("No matching memory db provider for %s", dbProvider)
	}
}

func (c *Provider) GetLogging() (LoggingProvider, error) {
	logProvider := c.Env.Spec.Logging.Provider
	switch logProvider {
	case "app-interface":
		return NewAppInterface(c)
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("No matching logging provider for %s", logProvider)
	}
}

func (c *Provider) SetUpEnvironment() error {
	var err error

	if _, err = c.GetObjectStore(); err != nil {
		return fmt.Errorf("setupenv: getobjectstore: %w", err)
	}

	if _, err = c.GetDatabase(); err != nil {
		return fmt.Errorf("setupenv: getdatabase: %w", err)
	}

	if _, err = c.GetKafka(); err != nil {
		return fmt.Errorf("setupenv: getkafka: %w", err)
	}

	if _, err = c.GetInMemoryDB(); err != nil {
		return fmt.Errorf("setupenv: getinmemorydb: %w", err)
	}

	if _, err = c.GetLogging(); err != nil {
		return fmt.Errorf("setupenv: getlogging: %w", err)
	}

	return nil
}
