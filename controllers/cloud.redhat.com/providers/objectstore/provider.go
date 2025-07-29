package objectstore

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// DefaultImageObjectStoreMinio defines the default MinIO object store image
var DefaultImageObjectStoreMinio = "minio/minio:RELEASE.2020-11-19T23-48-16Z-amd64"

// GetObjectStoreMinioImage returns the MinIO object store image for the environment
func GetObjectStoreMinioImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.ObjectStore.Images.Minio != "" {
		return env.Spec.Providers.ObjectStore.Images.Minio
	}
	if clowderconfig.LoadedConfig.Images.ObjectStoreMinio != "" {
		return clowderconfig.LoadedConfig.Images.ObjectStoreMinio
	}
	return DefaultImageObjectStoreMinio
}

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
		return nil, errors.NewClowderError(errStr)
	}
}

func init() {
	providers.ProvidersRegistration.Register(GetObjectStore, 5, ProvName)
}
