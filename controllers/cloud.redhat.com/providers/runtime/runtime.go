package runtime

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/database"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/inmemorydb"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/kafka"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/logging"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/objectstore"
)

func GetObjectStore(c *p.Provider) (p.ObjectStoreProvider, error) {
	objectStoreProvider := c.Env.Spec.ObjectStore.Provider
	switch objectStoreProvider {
	case "minio":
		return objectstore.NewMinIO(c)
	case "app-interface":
		return &objectstore.AppInterfaceObjectstoreProvider{Provider: *c}, nil
	default:
		errStr := fmt.Sprintf("No matching object store provider for %s", objectStoreProvider)
		return nil, errors.New(errStr)
	}
}

func GetDatabase(c *p.Provider) (p.DatabaseProvider, error) {
	dbProvider := c.Env.Spec.Database.Provider
	switch dbProvider {
	case "local":
		return database.NewLocalDBProvider(c)
	default:
		errStr := fmt.Sprintf("No matching db provider for %s", dbProvider)
		return nil, errors.New(errStr)
	}
}

func GetKafka(c *p.Provider) (p.KafkaProvider, error) {
	kafkaProvider := c.Env.Spec.Kafka.Provider
	switch kafkaProvider {
	case "operator":
		return kafka.NewStrimzi(c)
	case "local":
		return kafka.NewLocalKafka(c)
	default:
		errStr := fmt.Sprintf("No matching kafka provider for %s", kafkaProvider)
		return nil, errors.New(errStr)
	}
}

func GetInMemoryDB(c *p.Provider) (p.InMemoryDBProvider, error) {
	dbProvider := c.Env.Spec.InMemoryDB.Provider
	switch dbProvider {
	case "redis":
		return inmemorydb.NewLocalRedis(c)
	default:
		errStr := fmt.Sprintf("No matching in-memory db provider for %s", dbProvider)
		return nil, errors.New(errStr)
	}
}

func GetLogging(c *p.Provider) (p.LoggingProvider, error) {
	logProvider := c.Env.Spec.Logging.Provider
	switch logProvider {
	case "app-interface":
		return logging.NewAppInterfaceLogging(c)
	case "none":
		return nil, nil
	default:
		errStr := fmt.Sprintf("No matching logging provider for %s", logProvider)
		return nil, errors.New(errStr)
	}
}

func SetUpEnvironment(c *p.Provider) error {
	var err error

	if _, err = GetObjectStore(c); err != nil {
		return errors.Wrap("setupenv: getobjectstore", err)
	}

	if _, err = GetDatabase(c); err != nil {
		return errors.Wrap("setupenv: getdatabase", err)
	}

	if _, err = GetKafka(c); err != nil {
		return errors.Wrap("setupenv: getkafka", err)
	}

	if _, err = GetInMemoryDB(c); err != nil {
		return errors.Wrap("setupenv: getinmemorydb", err)
	}

	if _, err = GetLogging(c); err != nil {
		return errors.Wrap("setupenv: getlogging", err)
	}

	return nil
}
