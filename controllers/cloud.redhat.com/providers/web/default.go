package web

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type webProvider struct {
	providers.Provider
}

func NewWebProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CoreService,
		CoreEnvoyConfigMap,
		CoreEnvoySecret,
		CoreEnvoyCABundle,
	)
	return &webProvider{Provider: *p}, nil
}

func (web *webProvider) EnvProvide() error {
	if web.Env.Spec.Providers.Web.TLS.Enabled {
		if err := web.createCAConfigMap(); err != nil {
			return err
		}
	}
	return nil
}

func (web *webProvider) Provide(app *crd.ClowdApp) error {

	web.Config.WebPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	web.Config.PublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	privatePort := web.Env.Spec.Providers.Web.PrivatePort
	if privatePort == 0 {
		privatePort = 10000
	}
	web.Config.PrivatePort = utils.IntPtr(int(privatePort))

	if err := web.populateCA(); err != nil {
		return err
	}

	for _, deployment := range app.Spec.Deployments {
		d := deployment
		if err := makeService(web.Ctx, web.Client, web.Cache, &d, app, web.Env); err != nil {
			return err
		}
	}
	return nil
}

func (web *webProvider) createCAConfigMap() error {
	cm := &core.ConfigMap{}
	cmnn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-envoy-ca", web.Env.Name),
		Namespace: web.Env.Status.TargetNamespace,
	}

	if err := web.Cache.Create(CoreEnvoyCABundle, cmnn, cm); err != nil {
		return err
	}

	cm.Name = cmnn.Name
	cm.Namespace = cmnn.Namespace
	cm.ObjectMeta.OwnerReferences = []metav1.OwnerReference{web.Env.MakeOwnerReference()}
	utils.UpdateAnnotations(cm, map[string]string{
		"service.beta.openshift.io/inject-cabundle": "true",
	})

	if err := web.Cache.Update(CoreEnvoyCABundle, cm); err != nil {
		return err
	}
	return nil
}

func (web *webProvider) populateCA() error {
	if web.Env.Spec.Providers.Web.TLS.Enabled {
		cm := &core.ConfigMap{}
		cmnn := types.NamespacedName{
			Name:      fmt.Sprintf("%s-envoy-ca", web.Env.Name),
			Namespace: web.Env.Status.TargetNamespace,
		}

		if err := web.Client.Get(web.Ctx, cmnn, cm); err != nil {
			return err
		}

		if _, ok := cm.Data["service-ca.crt"]; !ok {
			return fmt.Errorf("could not get CA from secret")
		}
		web.Config.PublicPortCA = utils.StringPtr(cm.Data["service-ca.crt"])
	}
	return nil
}
