package objectstore

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ObjectStoreProvider is the interface for apps to use to configure object
// stores
type ObjectStoreProvider interface {
	p.Configurable
	CreateBuckets(app *crd.ClowdApp) error
}

func GetObjectStore(c *p.Provider) (ObjectStoreProvider, error) {
	objectStoreMode := c.Env.Spec.Providers.ObjectStore.Mode
	switch objectStoreMode {
	case "minio":
		return NewMinIO(c)
	case "app-interface":
		return &AppInterfaceObjectstoreProvider{Provider: *c}, nil
	default:
		errStr := fmt.Sprintf("No matching object store mode for %s", objectStoreMode)
		return nil, errors.New(errStr)
	}
}

func RunAppProvider(provider providers.Provider, c *config.AppConfig, app *crd.ClowdApp) error {
	if len(app.Spec.ObjectStore) == 0 {
		objectStoreProvider, err := GetObjectStore(&provider)

		if err != nil {
			return err
		}

		err = objectStoreProvider.CreateBuckets(app)

		if err != nil {
			return err
		}

		objectStoreProvider.Configure(c)
	}
	return nil
}

func RunEnvProvider(provider providers.Provider) error {
	_, err := GetObjectStore(&provider)

	if err != nil {
		return err
	}

	return nil
}
