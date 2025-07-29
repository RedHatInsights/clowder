package web

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "web"

// GetWeb returns the correct web provider.
func GetWeb(c *providers.Provider) (providers.ClowderProvider, error) {

	webMode := c.Env.Spec.Providers.Web.Mode
	switch webMode {
	case "none", "operator":
		return NewWebProvider(c)
	case "local":
		return NewLocalWebProvider(c)
	default:
		errStr := fmt.Sprintf("No matching web mode for %s", webMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	providers.ProvidersRegistration.Register(GetWeb, 1, ProvName)
}
