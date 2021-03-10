package database

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// ProvName is the providers name ident.
var ProvName = "database"

var imageList map[int32]string

// GetDatabase returns the correct database provider based on the environment.
func GetDatabase(c *p.Provider) (p.ClowderProvider, error) {
	dbMode := c.Env.Spec.Providers.Database.Mode
	switch dbMode {
	case "local":
		return NewLocalDBProvider(c)
	case "app-interface":
		return NewAppInterfaceDBProvider(c)
	case "none", "":
		return NewNoneDBProvider(c)
	default:
		errStr := fmt.Sprintf("No matching db mode for %s", dbMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetDatabase, 5, ProvName)
	imageList = map[int32]string{
		12: "quay.io/cloudservices/postgresql-rds:12-1",
		10: "quay.io/cloudservices/postgresql-rds:10-1",
	}
}

func checkDependency(app *crd.ClowdApp) error {
	for _, appName := range app.Spec.Dependencies {
		if app.Spec.Database.SharedDBAppName == appName {
			return nil
		}
	}

	return errors.New("The requested app's db was not found in the dependencies")
}
