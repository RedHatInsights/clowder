package database

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// DatabaseProvider is the interface for apps to use to configure databases
type DatabaseProvider interface {
	p.Configurable
	CreateDatabase(app *crd.ClowdApp) error
}

func GetDatabase(c *p.Provider) (DatabaseProvider, error) {
	dbProvider := c.Env.Spec.Database.Provider
	switch dbProvider {
	case "local":
		return NewLocalDBProvider(c)
	default:
		errStr := fmt.Sprintf("No matching db provider for %s", dbProvider)
		return nil, errors.New(errStr)
	}
}

func RunAppProvider(provider providers.Provider, c *config.AppConfig, app *crd.ClowdApp) error {
	dbSpec := app.Spec.Database

	if dbSpec.Name != "" {
		databaseProvider, err := GetDatabase(&provider)

		if err != nil {
			return errors.Wrap("Failed to init db provider", err)
		}

		err = databaseProvider.CreateDatabase(app)
		if err != nil {
			return err
		}
		databaseProvider.Configure(c)
	}
	return nil
}

func RunEnvProvider(provider providers.Provider) error {
	_, err := GetDatabase(&provider)

	if err != nil {
		return err
	}

	return nil
}
