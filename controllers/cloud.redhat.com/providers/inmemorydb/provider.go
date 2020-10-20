package inmemorydb

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// InMemoryDBProvider is the interface for apps to use to configure in-memory
// databases
type InMemoryDBProvider interface {
	p.Configurable
	CreateInMemoryDB(app *crd.ClowdApp) error
}

func GetInMemoryDB(c *p.Provider) (InMemoryDBProvider, error) {
	dbMode := c.Env.Spec.Providers.InMemoryDB.Mode
	switch dbMode {
	case "redis":
		return NewLocalRedis(c)
	default:
		errStr := fmt.Sprintf("No matching in-memory db mode for %s", dbMode)
		return nil, errors.New(errStr)
	}
}

func RunAppProvider(provider p.Provider, c *config.AppConfig, app *crd.ClowdApp) error {
	if app.Spec.InMemoryDB {
		inMemoryDbProvider, err := GetInMemoryDB(&provider)

		if err != nil {
			return errors.Wrap("Failed to init in-memory db provider", err)
		}

		err = inMemoryDbProvider.CreateInMemoryDB(app)
		if err != nil {
			return errors.Wrap("Failed to create in-memory db", err)
		}
		inMemoryDbProvider.Configure(c)
	}
	return nil
}

func RunEnvProvider(provider p.Provider) error {
	_, err := GetInMemoryDB(&provider)

	if err != nil {
		return err
	}

	return nil
}
