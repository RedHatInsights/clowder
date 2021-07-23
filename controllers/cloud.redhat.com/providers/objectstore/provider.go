package objectstore

import (
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName is the providers ident.
var ProvName = "objectstore"

// GetObjectStore returns the correct object store provider based on the environment.
func GetObjectStore(c *providers.Provider) (providers.ClowderProvider, error) {
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
	providers.ProvidersRegistration.Register(GetObjectStore, 5, ProvName)
}
