package inmemorydb

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName is the name/ident of the provider
var ProvName = "inmemorydb"

// GetInMemoryDB returns the correct in-memory DB provider based on the environment.
func GetInMemoryDB(c *providers.Provider) (providers.ClowderProvider, error) {
	dbMode := c.Env.Spec.Providers.InMemoryDB.Mode
	switch dbMode {
	case "redis":
		return NewLocalRedis(c)
	case "elasticache":
		return NewElasticache(c)
	case "none", "":
		return NewNoneInMemoryDb(c)
	default:
		errStr := fmt.Sprintf("No matching in-memory db mode for %s", dbMode)
		return nil, errors.NewClowderError(errStr)
	}
}

// Checks this app's list of dependencies to ensure shared app is included
func checkDependency(app *crd.ClowdApp) error {
	for _, appName := range app.Spec.Dependencies {
		if app.Spec.SharedInMemoryDBAppName == appName {
			return nil
		}
	}

	return errors.NewClowderError("The requested app's in memory db was not found in the dependencies")
}

func init() {
	providers.ProvidersRegistration.Register(GetInMemoryDB, 5, ProvName)
}
