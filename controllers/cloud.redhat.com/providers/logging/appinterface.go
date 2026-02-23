// Package logging provides logging configuration management for Clowder applications
package logging

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type appInterfaceLoggingProvider struct {
	providers.Provider
}

// NewAppInterfaceLogging returns a new app-interface logging provider object.
func NewAppInterfaceLogging(p *providers.Provider) providers.ClowderProvider {
	return &appInterfaceLoggingProvider{Provider: *p}
}

func (a *appInterfaceLoggingProvider) EnvProvide() error {
	return nil
}

func (a *appInterfaceLoggingProvider) Provide(app *crd.ClowdApp) error {
	a.Config.Logging = config.LoggingConfig{}
	return setCloudwatchSecret(app.Namespace, &a.Provider, &a.Config.Logging)
}

func setCloudwatchSecret(ns string, p *providers.Provider, c *config.LoggingConfig) error {

	name := types.NamespacedName{
		Name:      "cloudwatch",
		Namespace: ns,
	}

	secret := core.Secret{}
	err := p.Client.Get(p.Ctx, name, &secret)

	if err != nil {
		return errors.Wrap("Failed to fetch cloudwatch secret", err)
	}

	if _, err := p.HashCache.CreateOrUpdateObject(&secret, true); err != nil {
		return err
	}

	if err := p.HashCache.AddClowdObjectToObject(p.Env, &secret); err != nil {
		return err
	}

	c.Cloudwatch = &config.CloudWatchConfig{
		AccessKeyId:     string(secret.Data["aws_access_key_id"]),
		SecretAccessKey: string(secret.Data["aws_secret_access_key"]),
		Region:          string(secret.Data["aws_region"]),
		LogGroup:        string(secret.Data["log_group_name"]),
	}
	c.Type = "cloudwatch"

	return nil
}
