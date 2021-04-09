package logging

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// GetLogging returns the correct logging provider based on the environment.
func GetLogging(c *providers.Provider) (providers.ClowderProvider, error) {
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
	providers.ProvidersRegistration.Register(GetLogging, 5, "logging")
}
