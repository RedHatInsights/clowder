package dependencies

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type dependenciesProvider struct {
	p.Provider
}

// NewDependenciesProvider returns a new End provider run at the end of the provider set.
func NewDependenciesProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &dependenciesProvider{Provider: *p}, nil
}

func (dep *dependenciesProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	if err := dep.makeDependencies(app, c); err != nil {
		return err
	}
	return nil
}
