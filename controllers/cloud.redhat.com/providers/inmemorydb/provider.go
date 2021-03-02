package inmemorydb

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// GetInMemoryDB returns the correct in-memory DB provider based on the environment.
func GetInMemoryDB(c *p.Provider) (p.ClowderProvider, error) {
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
	p.ProvidersRegistration.Register(GetInMemoryDB, 1, "inmemorydb")
}
