// Package dependencies provides dependency management functionality for Clowder applications
package dependencies

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type dependenciesProvider struct {
	providers.Provider
}

// NewDependenciesProvider returns a new End provider run at the end of the provider set.
func NewDependenciesProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &dependenciesProvider{Provider: *p}, nil
}

func (dep *dependenciesProvider) EnvProvide() error {
	return nil
}

func (dep *dependenciesProvider) Provide(app *crd.ClowdApp) error {
	return dep.makeDependencies(app)
}
