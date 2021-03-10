package logging

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// GetLogging returns the correct logging provider based on the environment.
func GetLogging(c *p.Provider) (p.ClowderProvider, error) {
	logMode := c.Env.Spec.Providers.Logging.Mode
	switch logMode {
	case "app-interface":
		return NewAppInterfaceLogging(c)
	case "none", "null", "":
		return NewNoneLogging(c)
	default:
		errStr := fmt.Sprintf("No matching logging mode for %s", logMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetLogging, 5, "logging")
}
