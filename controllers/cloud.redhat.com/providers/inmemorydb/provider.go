package inmemorydb

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
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
		return nil, errors.New(errStr)
	}
}

func init() {
	providers.ProvidersRegistration.Register(GetInMemoryDB, 5, ProvName)
}
