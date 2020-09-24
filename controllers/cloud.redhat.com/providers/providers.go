package providers

import (
	"context"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProviderContext is just a wrapper for the parameters that need to be passed
// in to all providers for initialization
type ProviderContext struct {
	Client client.Client
	Ctx    context.Context
	Env    *crd.ClowdEnvironment
}

// Provider represents the common functions that all providers should have.
type Provider interface {
	// Configure is responsible for applying the respective section of
	// AppConfig.  It should be called in the app reconciler after all
	// provider-specific API calls (e.g. CreateBucket) have been made.
	Configure(c *config.AppConfig)

	// New is called to pass in a standard set of data to initialize providers
	// and also to create any environment-wide resources required to operate
	// the provider.
	New(ctx *ProviderContext) error
}

// ObjectStoreProvider is the interface for apps to use to configure object
// stores
type ObjectStoreProvider interface {
	Provider
	CreateBucket(bucket string) error
}

// KafkaProvider is the interface for apps to use to configure kafka topics
type KafkaProvider interface {
	CreateTopic(topic strimzi.KafkaTopic) error
}

// DatabaseProvider is the interface for apps to use to configure databases
type DatabaseProvider interface {
	CreateDatabase(name string) error
}

// InMemoryDBProvider is the interface for apps to use to configure in-memory
// databases
type InMemoryDBProvider interface {
	CreateInMemoryDB(name string) error
}

// LoggingProvider is the interface for apps to use to configure logging.  This
// may not be needed on a per-app basis; logging is often only configured on a
// per-environment basis.
type LoggingProvider interface {
	SetupLogging(name string) error
}

// ProviderChooser will return the correct Provider given the current
// environment settings

func (c *ProviderContext) GetObjectStore() (ObjectStoreProvider, error) {
	switch c.Ctx.Env.Spec.ObjectStore.Provider {
	case "minio":
		return NewMinIO(c.Ctx)
	}
}

func (c *ProviderContext) GetDatabase() (DatabaseProvider, error) {
	return nil, nil
}
func (c *ProviderContext) GetKafka() (KafkaProvider, error) {
	return nil, nil
}
func (c *ProviderContext) GetInMemoryDB() (InMemoryDBProvider, error) {
	return nil, nil
}
func (c *ProviderContext) GetLogging() (LoggingProvider, error) {
	return nil, nil
}
