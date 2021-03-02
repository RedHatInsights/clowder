package kafka

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneKafkaProvider struct {
	p.Provider
	Config config.DatabaseConfig
}

// NewNoneKafka returns a new non kafka provider object.
func NewNoneKafka(p *p.Provider) (providers.ClowderProvider, error) {
	return &noneKafkaProvider{Provider: *p}, nil
}

func (k *noneKafkaProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return nil
}
