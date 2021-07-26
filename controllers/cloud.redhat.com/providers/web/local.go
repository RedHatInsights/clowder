package web

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type mockConfig struct {
	BOPURL      string
	KeycloakURL string
}

type localWebProvider struct {
	providers.Provider
	config mockConfig
}

func NewLocalWebProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	wp := &localWebProvider{Provider: *p}

	objList := []providers.ResourceIdent{
		WebKeycloakDeployment,
		WebKeycloakService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "keycloak", makeKeycloak, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	objList = []providers.ResourceIdent{
		WebBOPDeployment,
		WebBOPService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "mbop", makeBOP, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	wp.config.BOPURL = fmt.Sprintf("http://%s-%s.%s.svc:8080", p.Env.GetClowdName(), "mbop", p.Env.GetClowdNamespace())
	wp.config.KeycloakURL = fmt.Sprintf("http://%s-%s.%s.svc:8080", p.Env.GetClowdName(), "keycloak", p.Env.GetClowdNamespace())

	nn := types.NamespacedName{
		Name:      "caddy-config",
		Namespace: p.Env.GetClowdNamespace(),
	}

	sec := &core.Secret{}
	if err := p.Cache.Create(WebSecret, nn, sec); err != nil {
		return nil, err
	}

	sec.Name = nn.Name
	sec.Namespace = nn.Namespace
	sec.ObjectMeta.OwnerReferences = []metav1.OwnerReference{p.Env.MakeOwnerReference()}
	sec.Type = core.SecretTypeOpaque

	sec.StringData = map[string]string{
		"bopurl":      wp.config.BOPURL,
		"keycloakurl": wp.config.KeycloakURL,
	}

	if err := p.Cache.Update(WebSecret, sec); err != nil {
		return nil, err
	}

	err := wp.configureKeycloak()

	if err != nil {
		newErr := errors.Wrap("couldn't config", err)
		newErr.Requeue = true
		return nil, newErr
	}

	return wp, nil

}

func (web *localWebProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	c.WebPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	c.PublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	privatePort := web.Env.Spec.Providers.Web.PrivatePort
	if privatePort == 0 {
		privatePort = 10000
	}
	c.PrivatePort = utils.IntPtr(int(privatePort))

	for _, deployment := range app.Spec.Deployments {

		if err := makeService(web.Cache, &deployment, app, web.Env); err != nil {
			return err
		}
	}

	c.Mock = &config.MockConfig{
		Bop:      providers.StrPtr(web.config.BOPURL),
		Keycloak: providers.StrPtr(web.config.KeycloakURL),
	}

	return nil
}
