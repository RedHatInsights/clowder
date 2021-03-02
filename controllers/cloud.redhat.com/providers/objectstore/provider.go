package objectstore

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// GetObjectStore returns the correct object store provider based on the environment.
func GetObjectStore(c *p.Provider) (p.ClowderProvider, error) {
	objectStoreMode := c.Env.Spec.Providers.ObjectStore.Mode
	switch objectStoreMode {
	case "minio":
		return NewMinIO(c)
	case "app-interface":
		return &appInterfaceObjectstoreProvider{Provider: *c}, nil
	case "none", "":
		return NewNoneObjectStore(c)
	default:
		errStr := fmt.Sprintf("No matching object store mode for %s", objectStoreMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetObjectStore, 1, "objectstore")
}
