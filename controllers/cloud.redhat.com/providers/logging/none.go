package logging

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type noneLoggingProvider struct {
	p.Provider
	Config config.LoggingConfig
}

// NewNoneLogging returns a new none logging provider object.
func NewNoneLogging(p *p.Provider) (providers.ClowderProvider, error) {
	provider := noneLoggingProvider{Provider: *p}

	return &provider, nil
}

func (a *noneLoggingProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
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
