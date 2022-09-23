package kafka

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneKafkaProvider struct {
	providers.Provider
}

// NewNoneKafka returns a new non kafka provider object.
func NewNoneKafka(p *providers.Provider) (providers.ClowderProvider, error) {
	return &noneKafkaProvider{Provider: *p}, nil
}

func (k *noneKafkaProvider) EnvProvide() error {
	return nil
}

func (k *noneKafkaProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
