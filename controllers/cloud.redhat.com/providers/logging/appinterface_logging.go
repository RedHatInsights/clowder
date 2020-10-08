package logging

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type AppInterfaceLoggingProvider struct {
	p.Provider
	Config config.LoggingConfig
}

func (a *AppInterfaceLoggingProvider) Configure(c *config.AppConfig) {
	c.Logging = a.Config
}

func NewAppInterfaceLogging(p *p.Provider) (p.LoggingProvider, error) {
	provider := AppInterfaceLoggingProvider{Provider: *p}

	return &provider, nil
}

func (a *AppInterfaceLoggingProvider) SetUpLogging(app *crd.ClowdApp) error {
	a.Config = config.LoggingConfig{}
	return setCloudwatchSecret(app.Namespace, &a.Provider, &a.Config)
}

func setCloudwatchSecret(ns string, p *p.Provider, c *config.LoggingConfig) error {

	name := types.NamespacedName{
		Name:      "cloudwatch",
		Namespace: ns,
	}

	secret := core.Secret{}
	fmt.Printf("%+v\n", p)
	err := p.Client.Get(p.Ctx, name, &secret)

	if err != nil {
		return errors.Wrap("Failed to fetch cloudwatch secret", err)
	}

	cwKeys := []string{
		"aws_access_key_id",
		"aws_secret_access_key",
		"aws_region",
		"log_group_name",
	}

	decoded := make([]string, 4)

	for i := 0; i < 4; i++ {
		decoded[i], err = utils.B64Decode(&secret, cwKeys[i])

		if err != nil {
			return errors.Wrap("Failed to b64 decode", err)
		}
	}

	c.Cloudwatch = &config.CloudWatchConfig{
		AccessKeyId:     decoded[0],
		SecretAccessKey: decoded[1],
		Region:          decoded[2],
		LogGroup:        decoded[3],
	}

	return nil
}
