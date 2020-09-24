package providers

import (
	"context"

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
	CreateInMemoryDB(name string) error
}

// LoggingProvider is the interface for apps to use to configure logging.  This
// may not be needed on a per-app basis; logging is often only configured on a
// per-environment basis.
type LoggingProvider interface {
	Configurable
	SetupLogging(name string) error
}

func (c *Provider) GetObjectStore() (ObjectStoreProvider, error) {
	var o ObjectStoreProvider
	var err error

	switch c.Env.Spec.ObjectStore.Provider {
	case "minio":
		o, err = NewMinIO(c)
	}

	return o, err
}

func (c *Provider) GetDatabase() (DatabaseProvider, error) {
	var o DatabaseProvider
	var err error

	switch c.Env.Spec.Database.Provider {
	case "local":
		o, err = NewLocalDBProvider(c)
	}

	return o, err
}

func (c *Provider) GetKafka() (KafkaProvider, error) {
	return nil, nil
}

func (c *Provider) GetInMemoryDB() (InMemoryDBProvider, error) {
	return nil, nil
}

func (c *Provider) GetLogging() (LoggingProvider, error) {
	return nil, nil
}
