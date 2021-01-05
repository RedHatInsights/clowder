package database

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

func GetDatabase(c *p.Provider) (p.ClowderProvider, error) {
	dbMode := c.Env.Spec.Providers.Database.Mode
	switch dbMode {
	case "local":
		return NewLocalDBProvider(c)
	case "app-interface":
		return NewAppInterfaceObjectstore(c)
	default:
		errStr := fmt.Sprintf("No matching db mode for %s", dbMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetDatabase, 1, "database")
}
