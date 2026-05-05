// Package reverseproxy provides a Clowder provider for deploying a reverse proxy in ephemeral environments.
package reverseproxy

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// DefaultImageReverseProxy defines the default reverse proxy image.
var DefaultImageReverseProxy = "quay.io/redhat-services-prod/hcc-platex-services-tenant/frontend-asset-proxy:latest"

// GetReverseProxyImage returns the reverse proxy image for the environment.
func GetReverseProxyImage(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.ReverseProxy.Images.Proxy != "" {
		return env.Spec.Providers.ReverseProxy.Images.Proxy
	}
	if clowderconfig.LoadedConfig.Images.ReverseProxy != "" {
		return clowderconfig.LoadedConfig.Images.ReverseProxy
	}
	return DefaultImageReverseProxy
}

// ProvName is the providers ident.
var ProvName = "reverseproxy"

// GetReverseProxy returns the correct reverse proxy provider based on the environment.
func GetReverseProxy(c *providers.Provider) (providers.ClowderProvider, error) {
	mode := c.Env.Spec.Providers.ReverseProxy.Mode
	switch mode {
	case "ephemeral":
		return NewLocalReverseProxy(c)
	case "none", "":
		return NewNoneReverseProxy(c)
	default:
		errStr := fmt.Sprintf("No matching reverse proxy mode for %s", mode)
		return nil, errors.NewClowderError(errStr)
	}
}

func init() {
	providers.ProvidersRegistration.Register(GetReverseProxy, 6, ProvName)
}
