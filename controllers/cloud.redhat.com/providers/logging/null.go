package logging

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type NullLoggingProvider struct {
	p.Provider
	Config config.LoggingConfig
}

func NewNullLogging(p *p.Provider) (providers.ClowderProvider, error) {
	provider := NullLoggingProvider{Provider: *p}

	return &provider, nil
}

func (a *NullLoggingProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	c.Logging = config.LoggingConfig{
		Cloudwatch: &config.CloudWatchConfig{
			AccessKeyId:     "",
			SecretAccessKey: "",
			Region:          "",
			LogGroup:        "",
		},
		Type: "null",
	}

	return nil
}
