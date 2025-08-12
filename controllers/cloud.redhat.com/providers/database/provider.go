package database

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName is the providers name ident.
var ProvName = "database"

// GetDatabase returns the correct database provider based on the environment.
func GetDatabase(c *p.Provider) (p.ClowderProvider, error) {
	dbMode := c.Env.Spec.Providers.Database.Mode
	switch dbMode {
	case "shared":
		return NewSharedDBProvider(c)
	case "local":
		return NewLocalDBProvider(c)
	case "app-interface":
		return NewAppInterfaceDBProvider(c)
	case "none", "":
		return NewNoneDBProvider(c)
	default:
		errStr := fmt.Sprintf("No matching db mode for %s", dbMode)
		return nil, errors.NewClowderError(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetDatabase, 5, ProvName)
}

func checkDependency(app *crd.ClowdApp) error {
	for _, appName := range app.Spec.Dependencies {
		if app.Spec.Database.SharedDBAppName == appName {
			return nil
		}
	}

	return errors.NewClowderError("The requested app's db was not found in the dependencies")
}
