package statefulset

import (
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

// ProvName sets the provider name identifier
var ProvName = "statefulSet"

// GetEnd returns the correct end provider.
func GetStatefulSet(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewStatefulSetProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetStatefulSet, 0, ProvName)
}
