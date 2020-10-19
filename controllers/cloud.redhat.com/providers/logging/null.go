package logging

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type NullLoggingProvider struct {
	p.Provider
	Config config.LoggingConfig
}

func (a *NullLoggingProvider) Configure(c *config.AppConfig) {
	c.Logging = a.Config
}

func NewNullLogging(p *p.Provider) (LoggingProvider, error) {
	provider := NullLoggingProvider{Provider: *p}

	return &provider, nil
}

func (a *NullLoggingProvider) SetUpLogging(app *crd.ClowdApp) error {
	a.Config = config.LoggingConfig{
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
