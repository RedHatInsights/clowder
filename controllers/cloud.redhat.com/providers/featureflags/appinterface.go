package featureflags

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type appInterfaceFeatureFlagProvider struct {
	providers.Provider
}

// NewAppInterfaceFeatureFlagsProvider creates a new app-interface feature flags provider.
func NewAppInterfaceFeatureFlagsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &appInterfaceFeatureFlagProvider{Provider: *p}, nil
}

func (ff *appInterfaceFeatureFlagProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	emptyNN := crd.NamespacedName{}
	if ff.Env.Spec.Providers.FeatureFlags.CredentialRef == emptyNN {
		return errors.New("no feature flag secret defined")
	}

	if ff.Env.Spec.Providers.FeatureFlags.Hostname == "" {
		return errors.New("hostname is not defined")
	}

	if ff.Env.Spec.Providers.FeatureFlags.Port == 0 {
		return errors.New("port is not defined")
	}

	sec := &core.Secret{}

	if err := ff.Client.Get(ff.Ctx, types.NamespacedName{
		Name:      ff.Env.Spec.Providers.FeatureFlags.CredentialRef.Name,
		Namespace: ff.Env.Spec.Providers.FeatureFlags.CredentialRef.Namespace,
	}, sec); err != nil {
		return err
	}

	accessToken, ok := sec.Data["CLIENT_ACCESS_TOKEN"]
	if !ok {
		return errors.New("Missing data")
	}

	stringAccessToken := string(accessToken)

	c.FeatureFlags = &config.FeatureFlagsConfig{
		ClientAccessToken: &stringAccessToken,
		Hostname:          ff.Env.Spec.Providers.FeatureFlags.Hostname,
		Port:              int(ff.Env.Spec.Providers.FeatureFlags.Port),
	}

	return nil
}
