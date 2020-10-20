package logging

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// LoggingProvider is the interface for apps to use to configure logging.  This
// may not be needed on a per-app basis; logging is often only configured on a
// per-environment basis.
type LoggingProvider interface {
	providers.Configurable
	SetUpLogging(app *crd.ClowdApp) error
}

func GetLogging(c *p.Provider) (LoggingProvider, error) {
	logMode := c.Env.Spec.Providers.Logging.Mode
	switch logMode {
	case "app-interface":
		return NewAppInterfaceLogging(c)
	case "none":
		return NewNullLogging(c)
	default:
		errStr := fmt.Sprintf("No matching logging mode for %s", logMode)
		return nil, errors.New(errStr)
	}
}

func RunAppProvider(provider providers.Provider, c *config.AppConfig, app *crd.ClowdApp) error {
	loggingProvider, err := GetLogging(&provider)

	if err != nil {
		return errors.Wrap("Failed to init logging provider", err)
	}

	if loggingProvider != nil {
		err = loggingProvider.SetUpLogging(app)

		if err != nil {
			return errors.Wrap("Failed to set up logging", err)
		}

		loggingProvider.Configure(c)
	}
	return nil
}

func RunEnvProvider(provider providers.Provider) error {
	_, err := GetLogging(&provider)

	if err != nil {
		return err
	}

	return nil
}
