package logging

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type noneLoggingProvider struct {
	providers.Provider
}

// NewNoneLogging returns a new none logging provider object.
func NewNoneLogging(p *providers.Provider) providers.ClowderProvider {
	return &noneLoggingProvider{Provider: *p}
}

func (a *noneLoggingProvider) EnvProvide() error {
	return nil
}

func (a *noneLoggingProvider) Provide(_ *crd.ClowdApp) error {
	a.Config.Logging = config.LoggingConfig{
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
