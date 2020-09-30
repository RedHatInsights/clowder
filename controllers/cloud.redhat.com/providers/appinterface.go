package providers

import (
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type AppInterfaceProvider struct {
	Provider
	Config config.LoggingConfig
}

func (a *AppInterfaceProvider) Configure(c *config.AppConfig) {
	c.Logging = a.Config
}

func NewAppInterface(p *Provider) (LoggingProvider, error) {
	provider := AppInterfaceProvider{Provider: *p}

	return &provider, nil
}

func (a *AppInterfaceProvider) CreateBucket(bucket string) error {
	return nil
}

func (a *AppInterfaceProvider) SetUpLogging(nn types.NamespacedName) error {
	a.Config = config.LoggingConfig{}
	return setCloudwatchSecret(nn, &a.Provider, &a.Config)
}

func setCloudwatchSecret(nn types.NamespacedName, p *Provider, c *config.LoggingConfig) error {

	name := types.NamespacedName{
		Name:      "cloudwatch",
		Namespace: nn.Namespace,
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
