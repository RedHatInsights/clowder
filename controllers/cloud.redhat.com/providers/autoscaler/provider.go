package autoscaler

import (
	"errors"
	"fmt"
	"log"

	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "autoscaler"

// GetAutoscaler returns the correct end provider.
func GetAutoScaler(c *p.Provider) (p.ClowderProvider, error) {

	autoMode := c.Env.Spec.Providers.AutoScaler.Mode

	log.Println("SimpleAutoScalerLOG: GetAutoScaler - autoMode is " + autoMode)

	switch autoMode {
	case "keda":
		return NewAutoScalerProvider(c)
	case "simple":
		return NewSimpleAutoScalerProvider(c)
	case "none", "":
		return NewNoneAutoScalerProvider(c)
	default:
		errStr := fmt.Sprintf("No matching autoscaler mode for %s", autoMode)
		return nil, errors.New(errStr)
	}
}

func init() {
	p.ProvidersRegistration.Register(GetAutoScaler, 10, ProvName)
}
