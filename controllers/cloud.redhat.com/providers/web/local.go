package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provDeploy "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// WebKeycloakDeployment is the mocked keycloak deployment
var WebKeycloakDeployment = rc.NewSingleResourceIdent(ProvName, "web_keycloak_deployment", &apps.Deployment{}, rc.ResourceOptions{WriteNow: true})

// WebKeycloakService is the mocked keycloak deployment
var WebKeycloakService = rc.NewSingleResourceIdent(ProvName, "web_keycloak_service", &core.Service{}, rc.ResourceOptions{WriteNow: true})

// WebKeycloakIngress is the keycloak ingress
var WebKeycloakIngress = rc.NewSingleResourceIdent(ProvName, "web_keycloak_ingress", &networking.Ingress{})

// WebKeycloakImportSecret is the keycloak import secret
var WebKeycloakImportSecret = rc.NewSingleResourceIdent(ProvName, "web_keycloak_import_secret", &core.Secret{})

// WebBOPDeployment is the mocked bop deployment
var WebBOPDeployment = rc.NewSingleResourceIdent(ProvName, "web_bop_deployment", &apps.Deployment{})

// WebKeycloakService is the mocked keycloak deployment
var WebBOPService = rc.NewSingleResourceIdent(ProvName, "web_bop_service", &core.Service{})

// WebBOPDeployment is the mocked bop deployment
var WebMocktitlementsDeployment = rc.NewSingleResourceIdent(ProvName, "web_mocktitlements_deployment", &apps.Deployment{})

// WebKeycloakService is the mocked keycloak deployment
var WebMocktitlementsService = rc.NewSingleResourceIdent(ProvName, "web_mocktitlements_service", &core.Service{})

// WebKeycloakIngress is the mocked bop ingress
var WebMocktitlementsIngress = rc.NewSingleResourceIdent(ProvName, "web_mocktitlements_ingress", &networking.Ingress{})

// WebSecret is the mocked secret config
var WebSecret = rc.NewMultiResourceIdent(ProvName, "web_secret", &core.Secret{})

// WebKeycloakSecret is the mocked secret config
var WebKeycloakSecret = rc.NewSingleResourceIdent(ProvName, "web_keycloak_secret", &core.Secret{}, rc.ResourceOptions{WriteNow: true})

// WebIngress is the mocked secret config
var WebIngress = rc.NewMultiResourceIdent(ProvName, "web_ingress", &networking.Ingress{})

type localWebProvider struct {
	providers.Provider
}

func NewLocalWebProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		WebKeycloakDeployment,
		WebKeycloakService,
		WebKeycloakIngress,
		WebKeycloakImportSecret,
		WebBOPDeployment,
		WebBOPService,
		WebMocktitlementsDeployment,
		WebMocktitlementsService,
		WebMocktitlementsIngress,
		WebSecret,
		WebKeycloakSecret,
		WebIngress,
	)
	return &localWebProvider{Provider: *p}, nil
}

func (web *localWebProvider) EnvProvide() error {
	if web.Env.Status.Hostname == "" {
		web.Env.Status.Hostname = web.Env.GenerateHostname(web.Ctx, web.Client, web.Log, !clowderconfig.LoadedConfig.Features.DisableRandomRoutes)
		err := web.Client.Status().Update(web.Ctx, web.Env)
		if err != nil {
			return err
		}
	}

	nn := providers.GetNamespacedName(web.Env, "keycloak")

	username := utils.RandString(8)

	password, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return errors.Wrap("couldn't generate password", err)
	}

	defaultPassword, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return errors.Wrap("couldn't generate defaultPassword", err)
	}

	dataInit := func() map[string]string {
		return map[string]string{
			"username":        username,
			"password":        password,
			"defaultUsername": "jdoe",
			"defaultPassword": defaultPassword,
			"version":         provutils.GetKeycloakVersion(web.Env),
			"bopurl":          fmt.Sprintf("http://%s-%s.%s.svc:8090", web.Env.GetClowdName(), "mbop", web.Env.GetClowdNamespace()),
		}
	}

	dataMap, err := providers.MakeOrGetSecret(web.Ctx, web.Env, web.Cache, WebKeycloakSecret, nn, dataInit)
	if err != nil {
		return errors.Wrap("couldn't set/get secret", err)
	}

	if err := setSecretVersion(web.Cache, nn, provutils.GetKeycloakVersion(web.Env)); err != nil {
		return errors.Wrap("couldn't set secret version", err)
	}

	objList := []rc.ResourceIdent{
		WebKeycloakDeployment,
		WebKeycloakService,
	}

	if err := providers.CachedMakeComponent(web.Cache, objList, web.Env, "keycloak", makeKeycloak, false, web.Env.IsNodePort()); err != nil {
		return err
	}

	if err := makeKeycloakImportSecretRealm(web.Cache, web.Env, (*dataMap)["defaultPassword"]); err != nil {
		return err
	}

	objList = []rc.ResourceIdent{
		WebBOPDeployment,
		WebBOPService,
	}

	if err := providers.CachedMakeComponent(web.Cache, objList, web.Env, "mbop", makeBOP, false, web.Env.IsNodePort()); err != nil {
		return err
	}

	objList = []rc.ResourceIdent{
		WebMocktitlementsDeployment,
		WebMocktitlementsService,
	}

	if err := providers.CachedMakeComponent(web.Cache, objList, web.Env, "mocktitlements", makeMocktitlements, false, web.Env.IsNodePort()); err != nil {
		return err
	}

	if err := makeMocktitlementsSecret(&web.Provider, web.Config); err != nil {
		return err
	}

	if err := makeMocktitlementsIngress(&web.Provider); err != nil {
		return err
	}

	if err := makeAuthIngress(&web.Provider); err != nil {
		return err
	}

	if err != nil {
		newErr := errors.Wrap("couldn't config", err)
		newErr.Requeue = true
		return newErr
	}

	return nil
}

func (web *localWebProvider) Provide(app *crd.ClowdApp) error {

	web.Config.WebPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	web.Config.PublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	privatePort := web.Env.Spec.Providers.Web.PrivatePort
	if privatePort == 0 {
		privatePort = 10000
	}
	web.Config.PrivatePort = utils.IntPtr(int(privatePort))

	for _, deployment := range app.Spec.Deployments {

		if err := makeService(web.Cache, &deployment, app, web.Env); err != nil {
			return err
		}

		if err := web.createIngress(app, &deployment); err != nil {
			return err
		}

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
			"bopurl":      fmt.Sprintf("http://%s-%s.%s.svc:8090", web.Env.GetClowdName(), "mbop", web.Env.GetClowdNamespace()),
			"keycloakurl": fmt.Sprintf("http://%s-%s.%s.svc:8080", web.Env.GetClowdName(), "keycloak", web.Env.GetClowdNamespace()),
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

		utils.UpdateAnnotations(&d.Spec.Template, annotations)

		if err := web.Cache.Update(provDeploy.CoreDeployment, d); err != nil {
			return err
		}

		if err := web.Cache.Update(WebSecret, sec); err != nil {
			return err
		}

	}

	return nil
}

func setSecretVersion(cache *rc.ObjectCache, nn types.NamespacedName, desiredVersion string) error {
	sec := &core.Secret{}
	if err := cache.Get(WebKeycloakSecret, sec, nn); err != nil {
		return errors.Wrap("couldn't get secret from cache", err)
	}

	if v, ok := sec.Data["version"]; !ok || string(v) != desiredVersion {
		if sec.StringData == nil {
			sec.StringData = map[string]string{}
		}
		sec.StringData["version"] = desiredVersion
	}

	if err := cache.Update(WebKeycloakSecret, sec); err != nil {
		return errors.Wrap("couldn't update secret in cache", err)
	}
	return nil
}

func makeMocktitlementsSecret(p *providers.Provider, cfg *config.AppConfig) error {
	nn := types.NamespacedName{
		Name:      "caddy-config-mocktitlements",
		Namespace: p.Env.GetClowdNamespace(),
	}

	sec := &core.Secret{}
	if err := p.Cache.Create(WebSecret, nn, sec); err != nil {
		return err
	}

	sec.Name = nn.Name
	sec.Namespace = nn.Namespace
	sec.ObjectMeta.OwnerReferences = []metav1.OwnerReference{p.Env.MakeOwnerReference()}
	sec.Type = core.SecretTypeOpaque

	envSec := &core.Secret{}
	envSecnn := providers.GetNamespacedName(p.Env, "keycloak")
	if err := p.Client.Get(p.Ctx, envSecnn, envSec); err != nil {
		return err
	}

	sec.StringData = map[string]string{
		"bopurl":      string(envSec.Data["bopurl"]),
		"keycloakurl": fmt.Sprintf("http://%s-%s.%s.svc:8080", p.Env.GetClowdName(), "keycloak", p.Env.GetClowdNamespace()),
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
	if err := p.Cache.Get(WebMocktitlementsDeployment, d, dnn); err != nil {
		return err
	}

	annotations := map[string]string{
		"clowder/authsidecar-confighash": hash,
	}

	utils.UpdateAnnotations(&d.Spec.Template, annotations)

	if err := p.Cache.Update(WebMocktitlementsDeployment, d); err != nil {
		return err
	}

	if err := p.Cache.Update(WebSecret, sec); err != nil {
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
							PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
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
							PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
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
							PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
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
