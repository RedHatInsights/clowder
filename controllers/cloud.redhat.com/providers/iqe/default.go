package iqe

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	// "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"fmt"
	// core "k8s.io/api/core/v1"

	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"k8s.io/apimachinery/pkg/types"
)

type iqeProvider struct {
	p.Provider
	Config config.IqeConfig
}

//
func NewIqeProvider(p *p.Provider) (providers.ClowderProvider, error) {
	iqe := &iqeProvider{Provider: *p, Config: config.IqeConfig{}}
	if p.Env.Spec.Providers.Iqe.ImageBase == "" {
		return nil, nil
	}

	// TODO : Override in invocation
	nn := types.NamespacedName{
		Name:      "iqe-config",
		Namespace: p.Env.Status.TargetNamespace,
	}

	iqeSettings := p.Env.Spec.Providers.Iqe
	iqeInit := func() map[string]string {
		return map[string]string{
			"imageBase":      iqeSettings.ImageBase,
			"k8sAccessLevel": fmt.Sprintf("%s", iqeSettings.K8SAccessLevel),
			"configAccess":   fmt.Sprintf("%s", iqeSettings.ConfigAccess),
		}
	}

	_, err := providers.MakeOrGetSecret(p.Ctx, p.Env, p.Cache, IqeSecret, nn, iqeInit)
	if err != nil {
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}

	iqe.Config = config.IqeConfig{
		ImageBase:      iqeSettings.ImageBase,
		K8SAccessLevel: fmt.Sprintf("%s", iqeSettings.K8SAccessLevel),
		ConfigAccess:   fmt.Sprintf("%s", iqeSettings.ConfigAccess),
	}

	// sa := &core.ServiceAccount{}
	// nn = types.NamespacedName{
	// 	Name:      "iqe",
	// 	Namespace: p.Env.Status.TargetNamespace,
	// }
	// sa.ImagePullSecrets = []core.LocalObjectReference{}

	// sa.ImagePullSecrets = append(sa.ImagePullSecrets, core.LocalObjectReference{
	// 	Name: "quay-cloudservices-pull",
	// })
	// update, err := utils.UpdateOrErr(p.Client.Get(p.Ctx, nn, sa))

	// if err != nil {
	// 	return nil, err
	// }

	// if err = update.Apply(p.Ctx, p.Client, sa); err != nil {
	// 	return nil, err
	// }
	// if err := p.Client.Create(p.Ctx, sa); err != nil {
	// 	return nil, err
	// }

	return iqe, nil
}

func (ip *iqeProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	c.Iqe = &ip.Config
	return nil
}
