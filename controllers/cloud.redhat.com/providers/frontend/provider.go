package frontend

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// FrontendProvider is the interface for apps to use to configure in-memory
// databases
type FrontendProvider interface {
	p.Configurable
	CreateFrontend(spec *crd.ClowdApp) error
}

func GetFrontend(c *p.Provider) (FrontendProvider, error) {
	webMode := c.Env.Spec.Providers.Web.Mode
	switch webMode {
	case "operator":
		return NewChromeFrontend(c)
	default:
		errStr := fmt.Sprintf("No matching frontend mode for %s", webMode)
		return nil, errors.New(errStr)
	}
}

func RunAppProvider(provider p.Provider, c *config.AppConfig, app *crd.ClowdApp) error {

	frontendProvider, err := GetFrontend(&provider)

	if err != nil {
		return errors.Wrap("Failed to init frontend provider", err)
	}

	err = frontendProvider.CreateFrontend(app)
	if err != nil {
		return errors.Wrap("Failed to create frontend", err)
	}
	frontendProvider.Configure(c)
	createConfigMap(app, &provider, c)
	return nil
}

func RunEnvProvider(provider p.Provider) error {
	_, err := GetFrontend(&provider)

	if err != nil {
		return err
	}

	return nil
}
