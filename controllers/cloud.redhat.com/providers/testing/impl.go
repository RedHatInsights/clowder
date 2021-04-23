package testing

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
)

func (t *testingProvider) MakeTestingConfig(app *crd.ClowdApp, c *config.AppConfig) error {
	testingSettings := t.Env.Spec.Providers.Testing

	iqeSettings := t.Env.Spec.Providers.Testing.Iqe
	t.Config = config.TestingConfig{
		K8SAccessLevel: string(testingSettings.K8SAccessLevel),
		ConfigAccess:   string(testingSettings.ConfigAccess),
		Iqe: &config.IqeConfig{
			ImageBase: iqeSettings.ImageBase,
		},
	}
	return nil
}
