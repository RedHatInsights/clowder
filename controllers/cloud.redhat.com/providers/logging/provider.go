package logging

import (
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// GetLogging returns the correct logging provider based on the environment.
func GetLogging(c *providers.Provider) (providers.ClowderProvider, error) {
	logMode := c.Env.Spec.Providers.Logging.Mode
	switch logMode {
	case "app-interface":
		return NewAppInterfaceLogging(c), nil
	case "none", "null", "":
		return NewNoneLogging(c), nil
	default:
		errStr := fmt.Sprintf("No matching logging mode for %s", logMode)
		return nil, errors.NewClowderError(errStr)
	}
}

func init() {
	providers.ProvidersRegistration.Register(GetLogging, 5, "logging")
}
