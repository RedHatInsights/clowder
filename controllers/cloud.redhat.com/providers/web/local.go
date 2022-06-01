package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provDeploy "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

type KeycloakConfig struct {
	URL             string
	Username        string
	Password        string
	DefaultUsername string
	DefaultPassword string
}

type backendConfig struct {
	BOPURL         string
	KeycloakConfig KeycloakConfig
}

type localWebProvider struct {
	providers.Provider
	config backendConfig
}

func NewLocalWebProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	if p.Env.Status.Hostname == "" {
		p.Env.Status.Hostname = p.Env.GenerateHostname(p.Ctx, p.Client, p.Log)
		err := p.Client.Status().Update(p.Ctx, p.Env)
		if err != nil {
			return nil, err
		}
	}

	wp := &localWebProvider{Provider: *p}

	nn := providers.GetNamespacedName(p.Env, "keycloak")

	dataInit := func() map[string]string {
		username := clowderconfig.LoadedConfig.Credentials.Keycloak.Username
		if username == "" {
			username = utils.RandString(8)
		}

		password := clowderconfig.LoadedConfig.Credentials.Keycloak.Password
		if password == "" {
			password = utils.RandString(8)
		}

		version := KEYCLOAK_VERSION

		defaultPassword := utils.RandString(8)

		return map[string]string{
			"username":        username,
			"password":        password,
			"defaultUsername": "jdoe",
			"defaultPassword": defaultPassword,
			"version":         version,
		}
	}

	dataMap, err := providers.MakeOrGetSecret(wp.Ctx, p.Env, wp.Cache, WebKeycloakSecret, nn, dataInit)
	if err != nil {
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}

	wp.config.KeycloakConfig.Username = (*dataMap)["username"]
	wp.config.KeycloakConfig.Password = (*dataMap)["password"]
	wp.config.KeycloakConfig.DefaultUsername = (*dataMap)["defaultUsername"]
	wp.config.KeycloakConfig.DefaultPassword = (*dataMap)["defaultPassword"]

	wp.config.BOPURL = fmt.Sprintf("http://%s-%s.%s.svc:8090", wp.Env.GetClowdName(), "mbop", wp.Env.GetClowdNamespace())
	wp.config.KeycloakConfig.URL = fmt.Sprintf("http://%s-%s.%s.svc:8080", wp.Env.GetClowdName(), "keycloak", wp.Env.GetClowdNamespace())

	objList := []rc.ResourceIdent{
		WebKeycloakDeployment,
		WebKeycloakService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "keycloak", makeKeycloak, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	if err := makeKeycloakImportSecretRealm(p.Cache, p.Env, wp.config.KeycloakConfig.DefaultPassword); err != nil {
		return nil, err
	}

	objList = []rc.ResourceIdent{
		WebBOPDeployment,
		WebBOPService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "mbop", makeBOP, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	objList = []rc.ResourceIdent{
		WebMocktitlementsDeployment,
		WebMocktitlementsService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "mocktitlements", makeMocktitlements, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	if err := makeMocktitlementsSecret(p, wp); err != nil {
		return nil, err
	}

	if err := makeMocktitlementsIngress(p); err != nil {
		return nil, err
	}

	if err := makeAuthIngress(p); err != nil {
		return nil, err
	}

	if err != nil {
		newErr := errors.Wrap("couldn't config", err)
		newErr.Requeue = true
		return nil, newErr
	}

	return wp, nil

}

func makeMocktitlementsSecret(p *providers.Provider, web *localWebProvider) error {
	nn := types.NamespacedName{
		Name:      "caddy-config-mocktitlements",
		Namespace: p.Env.GetClowdNamespace(),
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
		"bopurl":      web.config.BOPURL,
		"keycloakurl": web.config.KeycloakConfig.URL,
		"whitelist":   "",
	}

	jsonData, err := json.Marshal(sec.StringData)
	if err != nil {
		return errors.Wrap("Failed to marshal config JSON", err)
	}

	h := sha256.New()
	h.Write([]byte(jsonData))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	d := &apps.Deployment{}
	dnn := providers.GetNamespacedName(p.Env, "mbop")
	if err := web.Cache.Get(WebMocktitlementsDeployment, d, dnn); err != nil {
		return err
	}

	annotations := map[string]string{
		"clowder/authsidecar-confighash": hash,
	}

	utils.UpdatePodTemplateAnnotations(&d.Spec.Template, annotations)

	if err := web.Cache.Update(WebMocktitlementsDeployment, d); err != nil {
		return err
	}

	if err := web.Cache.Update(WebSecret, sec); err != nil {
		return err
	}
	return nil
}

func makeMocktitlementsIngress(p *providers.Provider) error {
	netobj := &networking.Ingress{}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-mocktitlements", p.Env.Name),
		Namespace: p.Env.Status.TargetNamespace,
	}

	if err := p.Cache.Create(WebMocktitlementsIngress, nn, netobj); err != nil {
		return err
	}

	labels := p.Env.GetLabels()
	labler := utils.MakeLabeler(nn, labels, p.Env)
	labler(netobj)

	ingressClass := p.Env.Spec.Providers.Web.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	netobj.Spec = networking.IngressSpec{
		TLS: []networking.IngressTLS{{
			Hosts: []string{},
		}},
		IngressClassName: &ingressClass,
		Rules: []networking.IngressRule{
			{
				Host: p.Env.Status.Hostname,
				IngressRuleValue: networking.IngressRuleValue{
					HTTP: &networking.HTTPIngressRuleValue{
						Paths: []networking.HTTPIngressPath{{
							Path:     "/api/entitlements/",
							PathType: (*networking.PathType)(common.StringPtr("Prefix")),
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: fmt.Sprintf("%s-mocktitlements", p.Env.Name),
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

	if err := p.Cache.Update(WebMocktitlementsIngress, netobj); err != nil {
		return err
	}
	return nil
}

func makeAuthIngress(p *providers.Provider) error {
	netobj := &networking.Ingress{}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-auth", p.Env.Name),
		Namespace: p.Env.Status.TargetNamespace,
	}

	if err := p.Cache.Create(WebKeycloakIngress, nn, netobj); err != nil {
		return err
	}

	labels := p.Env.GetLabels()
	labler := utils.MakeLabeler(nn, labels, p.Env)
	labler(netobj)

	ingressClass := p.Env.Spec.Providers.Web.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	netobj.Spec = networking.IngressSpec{
		TLS: []networking.IngressTLS{{
			Hosts: []string{},
		}},
		IngressClassName: &ingressClass,
		Rules: []networking.IngressRule{
			{
				Host: getAuthHostname(p.Env.Status.Hostname),
				IngressRuleValue: networking.IngressRuleValue{
					HTTP: &networking.HTTPIngressRuleValue{
						Paths: []networking.HTTPIngressPath{{
							Path:     "/",
							PathType: (*networking.PathType)(common.StringPtr("Prefix")),
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: fmt.Sprintf("%s-keycloak", p.Env.Name),
									Port: networking.ServiceBackendPort{
										Name: "keycloak",
									},
								},
							},
						}},
					},
				},
			},
		},
	}

	if err := p.Cache.Update(WebKeycloakIngress, netobj); err != nil {
		return err
	}
	return nil
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

		web.createIngress(app, &deployment)

		c.BOPURL = providers.StrPtr(web.config.BOPURL)

		nn := types.NamespacedName{
			Name:      fmt.Sprintf("caddy-config-%s-%s", app.Name, deployment.Name),
			Namespace: app.Namespace,
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
			"bopurl":      web.config.BOPURL,
			"keycloakurl": web.config.KeycloakConfig.URL,
			"whitelist":   strings.Join(deployment.WebServices.Public.WhitelistPaths, ","),
		}

		jsonData, err := json.Marshal(sec.StringData)
		if err != nil {
			return errors.Wrap("Failed to marshal config JSON", err)
		}

		h := sha256.New()
		h.Write([]byte(jsonData))
		hash := fmt.Sprintf("%x", h.Sum(nil))

		d := &apps.Deployment{}
		dnn := app.GetDeploymentNamespacedName(&deployment)
		if err := web.Cache.Get(provDeploy.CoreDeployment, d, dnn); err != nil {
			return err
		}

		annotations := map[string]string{
			"clowder/authsidecar-confighash": hash,
		}

		utils.UpdatePodTemplateAnnotations(&d.Spec.Template, annotations)

		if err := web.Cache.Update(provDeploy.CoreDeployment, d); err != nil {
			return err
		}

		if err := web.Cache.Update(WebSecret, sec); err != nil {
			return err
		}

	}

	return nil
}

func (web *localWebProvider) createIngress(app *crd.ClowdApp, deployment *crd.Deployment) error {

	if !deployment.WebServices.Public.Enabled && !bool(deployment.Web) {
		return nil
	}

	netobj := &networking.Ingress{}

	nn := app.GetDeploymentNamespacedName(deployment)

	if err := web.Cache.Create(WebIngress, nn, netobj); err != nil {
		return err
	}

	labels := app.GetLabels()
	labler := utils.MakeLabeler(nn, labels, app)
	labler(netobj)

	ingressClass := web.Env.Spec.Providers.Web.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	apiPath := deployment.WebServices.Public.ApiPath

	if apiPath == "" {
		apiPath = nn.Name
	}

	netobj.Spec = networking.IngressSpec{
		TLS: []networking.IngressTLS{{
			Hosts: []string{},
		}},
		IngressClassName: &ingressClass,
		Rules: []networking.IngressRule{
			{
				Host: web.Env.Status.Hostname,
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

	return nil
}

func getAuthHostname(hostname string) string {
	hostComponents := strings.Split(hostname, ".")
	hostComponents[0] = hostComponents[0] + "-auth"
	return strings.Join(hostComponents, ".")
}
