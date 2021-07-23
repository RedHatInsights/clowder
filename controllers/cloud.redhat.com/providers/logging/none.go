package logging

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneLoggingProvider struct {
	providers.Provider
	Config config.LoggingConfig
}

// NewNoneLogging returns a new none logging provider object.
func NewNoneLogging(p *providers.Provider) (providers.ClowderProvider, error) {
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
