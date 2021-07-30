package web

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type backendConfig struct {
	BOPURL      string
	KeycloakURL string
}

type localWebProvider struct {
	providers.Provider
	config backendConfig
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

	web.createIngress(app)

	BOPURL := fmt.Sprintf("http://%s-%s.%s.svc:8080", web.Env.GetClowdName(), "mbop", web.Env.GetClowdNamespace())
	KeycloakURL := fmt.Sprintf("http://%s-%s.%s.svc:8080", web.Env.GetClowdName(), "keycloak", web.Env.GetClowdNamespace())

	c.BOPURL = providers.StrPtr(web.config.BOPURL)

	nn := types.NamespacedName{
		Name:      "caddy-config",
		Namespace: web.Env.GetClowdNamespace(),
	}

	sec := &core.Secret{}
	if err := web.Cache.Create(WebSecret, nn, sec); err != nil {
		return err
	}

	sec.Name = nn.Name
	sec.Namespace = nn.Namespace
	sec.ObjectMeta.OwnerReferences = []metav1.OwnerReference{web.Env.MakeOwnerReference()}
	sec.Type = core.SecretTypeOpaque

	sec.StringData = map[string]string{
		"bopurl":      BOPURL,
		"keycloakurl": KeycloakURL,
	}

	if err := web.Cache.Update(WebSecret, sec); err != nil {
		return err
	}

	return nil
}

func (web *localWebProvider) createIngress(app *crd.ClowdApp) error {

	for _, deployment := range app.Spec.Deployments {

		if !deployment.WebServices.Public.Enabled && !bool(deployment.Web) {
			continue
		}

		netobj := &networking.Ingress{}

		nn := app.GetDeploymentNamespacedName(&deployment)

		if err := web.Cache.Create(WebIngress, nn, netobj); err != nil {
			return err
		}

		labels := app.GetLabels()
		labler := utils.MakeLabeler(nn, labels, app)
		labler(netobj)

		apiPath := deployment.WebServices.Public.ApiPath

		if apiPath == "" {
			apiPath = nn.Name
		}

		netobj.Spec = networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: web.Env.Name,
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{{
								Path:     fmt.Sprintf("/api/%s/", apiPath),
								PathType: (*networking.PathType)(common.StringPtr("Prefix")),
								Backend: networking.IngressBackend{
									Service: &networking.IngressServiceBackend{
										Name: nn.Name,
										Port: networking.ServiceBackendPort{
											Name: "auth",
										},
									},
								},
							}},
						},
					},
				},
			},
		}

		if err := web.Cache.Update(WebIngress, netobj); err != nil {
			return err
		}
	}

	return nil
}
