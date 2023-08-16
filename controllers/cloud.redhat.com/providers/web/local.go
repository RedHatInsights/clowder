package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provDeploy "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
	acme "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
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
var WebGatewayDeployment = rc.NewSingleResourceIdent(ProvName, "web_gateway_deployment", &apps.Deployment{})

// WebIngress is the mocked secret config
var WebGatewayIngress = rc.NewSingleResourceIdent(ProvName, "web_gateway_ingress", &networking.Ingress{})

// WebKeycloakService is the mocked keycloak deployment
var WebGatewayService = rc.NewSingleResourceIdent(ProvName, "web_gateway_service", &core.Service{})

// WebKeycloakService is the mocked keycloak deployment
var WebGatewayConfigMap = rc.NewSingleResourceIdent(ProvName, "web_gateway_configmap", &core.Service{})

// WebKeycloakService is the mocked keycloak deployment
var WebGatewayCertificateIssuer = rc.NewSingleResourceIdent(ProvName, "web_gateway_cert_issuer", &certmanager.Issuer{})

// WebKeycloakService is the mocked keycloak deployment
var WebGatewayCertificate = rc.NewSingleResourceIdent(ProvName, "web_gateway_certificate", &certmanager.Certificate{})

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
		WebGatewayDeployment,
		WebGatewayIngress,
		WebGatewayService,
		WebGatewayConfigMap,
		WebGatewayCertificate,
		WebGatewayCertificateIssuer,
		CoreEnvoyConfigMap,
		CoreService,
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

	dataMap, err := providers.MakeOrGetSecret(web.Env, web.Cache, WebKeycloakSecret, nn, dataInit)
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

	if err := makeMocktitlementsSecret(&web.Provider); err != nil {
		return err
	}

	if err := makeMocktitlementsIngress(&web.Provider); err != nil {
		return err
	}

	if err := makeWebGatewayIngress(&web.Provider); err != nil {
		return err
	}

	if err := makeWebGatewayConfigMap(&web.Provider); err != nil {
		return err
	}

	if err := makeWebGatewayCertificateIssuer(&web.Provider); err != nil {
		return err
	}

	if err := makeWebGatewayCertificate(&web.Provider); err != nil {
		return err
	}

	objList = []rc.ResourceIdent{
		WebGatewayDeployment,
		WebGatewayService,
	}

	if err := providers.CachedMakeComponent(web.Cache, objList, web.Env, "caddy-gateway", makeWebGatewayDeployment, false, web.Env.IsNodePort()); err != nil {
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

	if err := web.populateCA(); err != nil {
		return err
	}

	for _, deployment := range app.Spec.Deployments {
		innerDeployment := deployment
		if err := makeService(web.Cache, &innerDeployment, app, web.Env); err != nil {
			return err
		}

		nn := types.NamespacedName{
			Name:      fmt.Sprintf("caddy-config-%s-%s", app.Name, innerDeployment.Name),
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
			"whitelist":   strings.Join(innerDeployment.WebServices.Public.WhitelistPaths, ","),
		}

		jsonData, err := json.Marshal(sec.StringData)
		if err != nil {
			return errors.Wrap("Failed to marshal config JSON", err)
		}

		h := sha256.New()
		h.Write([]byte(jsonData))
		hash := fmt.Sprintf("%x", h.Sum(nil))

		d := &apps.Deployment{}
		dnn := app.GetDeploymentNamespacedName(&innerDeployment)
		if err := web.Cache.Get(provDeploy.CoreDeployment, d, dnn); err != nil {
			return err
		}

		if web.Env.Spec.Providers.Web.TLS.Enabled {
			provutils.AddCertVolume(&d.Spec.Template.Spec, dnn.Name)
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

func (web *localWebProvider) populateCA() error {
	if web.Env.Spec.Providers.Web.TLS.Enabled {
		web.Config.TlsCAPath = utils.StringPtr("/cdapp/certs/openshift-service-ca.crt")
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

func makeMocktitlementsSecret(p *providers.Provider) error {
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

	return p.Cache.Update(WebSecret, sec)
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

	return p.Cache.Update(WebMocktitlementsIngress, netobj)
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

	return p.Cache.Update(WebKeycloakIngress, netobj)
}

func makeWebGatewayIngress(p *providers.Provider) error {
	netobj := &networking.Ingress{}

	nn := providers.GetNamespacedName(p.Env, "caddy-gateway")

	if err := p.Cache.Create(WebGatewayIngress, nn, netobj); err != nil {
		return err
	}

	labels := p.Env.GetLabels()
	labler := utils.MakeLabeler(nn, labels, p.Env)
	labler(netobj)

	//TODO: only for nginx
	utils.UpdateAnnotations(netobj, map[string]string{
		"nginx.ingress.kubernetes.io/ssl-passthrough":  "true",
		"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
		//"kubernetes.io/ingress.class":                  "nginx",
		// "nginx.ingress.kubernetes.io/ssl-redirect": "true",
	})

	ingressClass := p.Env.Spec.Providers.Web.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	netobj.Spec = networking.IngressSpec{
		// TLS: []networking.IngressTLS{{
		// 	Hosts: []string{},
		// }},
		IngressClassName: &ingressClass,
		Rules: []networking.IngressRule{
			{
				Host: p.Env.Status.Hostname,
				IngressRuleValue: networking.IngressRuleValue{
					HTTP: &networking.HTTPIngressRuleValue{
						Paths: []networking.HTTPIngressPath{{
							Path:     "/api/",
							PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: nn.Name,
									Port: networking.ServiceBackendPort{
										// Name: "gateway",
										Number: 9090,
									},
								},
							},
						}},
						// }, {
						// 	Path:     "/v1/",
						// 	PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
						// 	Backend: networking.IngressBackend{
						// 		Service: &networking.IngressServiceBackend{
						// 			Name: nn.Name,
						// 			Port: networking.ServiceBackendPort{
						// 				Name: "gateway",
						// 			},
						// 		},
						// 	},
						// }},
					},
				},
			},
		},
	}

	return p.Cache.Update(WebGatewayIngress, netobj)
}

func getAuthHostname(hostname string) string {
	hostComponents := strings.Split(hostname, ".")
	hostComponents[0] += "-auth"
	return strings.Join(hostComponents, ".")
}

func makeWebGatewayCertificate(p *providers.Provider) error {
	certi := &certmanager.Certificate{}

	nn := types.NamespacedName{
		Name:      "caddy-gateway",
		Namespace: p.Env.GetClowdNamespace(),
	}

	if err := p.Cache.Create(WebGatewayCertificate, nn, certi); err != nil {
		return err
	}

	labels := p.Env.GetLabels()
	labler := utils.MakeLabeler(nn, labels, p.Env)
	labler(certi)

	if p.Env.Spec.Providers.Web.GatewayCertMode == "acme" {
		certi.Spec = *acmeCert(p)
	} else {
		certi.Spec = *selfSignedCert(p)
	}
	return p.Cache.Update(WebGatewayCertificate, certi)
}

func makeWebGatewayCertificateIssuer(p *providers.Provider) error {
	certi := &certmanager.Issuer{}

	nn := types.NamespacedName{
		Name:      p.Env.GetClowdNamespace(),
		Namespace: p.Env.GetClowdNamespace(),
	}

	if err := p.Cache.Create(WebGatewayCertificateIssuer, nn, certi); err != nil {
		return err
	}

	labels := p.Env.GetLabels()
	labler := utils.MakeLabeler(nn, labels, p.Env)
	labler(certi)

	if p.Env.Spec.Providers.Web.GatewayCertMode == "acme" {
		certi.Spec = *acmeIssuerSpec(p)
	} else {
		certi.Spec = *selfSignedIssuerSpec(p)
	}

	return p.Cache.Update(WebGatewayCertificateIssuer, certi)
}

func acmeIssuerSpec(p *providers.Provider) *certmanager.IssuerSpec {
	return &certmanager.IssuerSpec{
		IssuerConfig: certmanager.IssuerConfig{
			ACME: &acme.ACMEIssuer{
				Email:          "psavage@redhat.com",
				PreferredChain: "",
				PrivateKey: v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: fmt.Sprintf("%s-cluster-issuer", p.Env.GetClowdNamespace()),
					},
				},
				Server: "https://acme-staging-v02.api.letsencrypt.org/directory",
				Solvers: []acme.ACMEChallengeSolver{{
					HTTP01: &acme.ACMEChallengeSolverHTTP01{
						Ingress: &acme.ACMEChallengeSolverHTTP01Ingress{},
					},
				}},
			},
		},
	}
}

func selfSignedIssuerSpec(p *providers.Provider) *certmanager.IssuerSpec {
	return &certmanager.IssuerSpec{
		IssuerConfig: certmanager.IssuerConfig{
			SelfSigned: &certmanager.SelfSignedIssuer{},
		},
	}
}

func acmeCert(p *providers.Provider) *certmanager.CertificateSpec {
	return &certmanager.CertificateSpec{
		DNSNames: []string{
			p.Env.Status.Hostname,
		},
		IssuerRef: v1.ObjectReference{
			Group: "cert-manager.io",
			Kind:  "Issuer",
			Name:  p.Env.GetClowdNamespace(),
		},
	}
}

func selfSignedCert(p *providers.Provider) *certmanager.CertificateSpec {
	return &certmanager.CertificateSpec{
		DNSNames: []string{
			p.Env.Status.Hostname,
		},
		IssuerRef: v1.ObjectReference{
			Group: "cert-manager.io",
			Kind:  "Issuer",
			Name:  p.Env.GetClowdNamespace(),
		},
		PrivateKey: &certmanager.CertificatePrivateKey{
			Algorithm: certmanager.ECDSAKeyAlgorithm,
			Size:      256,
		},
		SecretName: providers.GetNamespacedName(p.Env, "caddy-gateway").Name,
	}
}

func makeWebGatewayConfigMap(p *providers.Provider) error {
	cm := &core.ConfigMap{}
	snn := providers.GetNamespacedName(p.Env, "caddy-gateway")

	if err := p.Cache.Create(CoreEnvoyConfigMap, snn, cm); err != nil {
		return err
	}

	cm.Name = snn.Name
	cm.Namespace = snn.Namespace
	cm.ObjectMeta.OwnerReferences = []metav1.OwnerReference{p.Env.MakeOwnerReference()}

	appList, err := p.Env.GetAppsInEnv(p.Ctx, p.Client)
	if err != nil {
		return err
	}

	whitelistStrings := []string{}
	upstreamList := []ProxyRoute{}
	for _, app := range appList.Items {
		innerApp := app
		for _, deployment := range innerApp.Spec.Deployments {
			innerDeployment := deployment
			whitelistStrings = append(whitelistStrings, innerDeployment.WebServices.Public.WhitelistPaths...)

			if !innerDeployment.WebServices.Public.Enabled && !bool(innerDeployment.Web) {
				continue
			}

			apiPath := innerDeployment.WebServices.Public.APIPath

			if apiPath == "" {
				apiPath = innerDeployment.Name
			}

			upstreamList = append(upstreamList, ProxyRoute{
				Upstream: fmt.Sprintf("http://%s:8000", innerDeployment.Name),
				Path:     apiPath,
			})
		}
	}

	bopHostname := fmt.Sprintf("http://%s-%s.%s.svc:8090", p.Env.GetClowdName(), "mbop", p.Env.GetClowdNamespace())

	upstreamList = append(upstreamList, ProxyRoute{
		Upstream: bopHostname,
		Path:     "/v1/registrations/*",
	})

	cmData, err := GenerateConfig(p.Env.Status.Hostname, bopHostname, whitelistStrings, upstreamList)
	if err != nil {
		return err
	}

	cm.Data = map[string]string{
		"Caddyfile.json": cmData,
	}

	return p.Cache.Update(CoreEnvoyConfigMap, cm)
}

func makeWebGatewayDeployment(o obj.ClowdObject, objMap providers.ObjectMap, _ bool, _ bool) {
	nn := providers.GetNamespacedName(o, "caddy-gateway")

	dd := objMap[WebGatewayDeployment].(*apps.Deployment)
	svc := objMap[WebGatewayService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	port := int32(9090)

	ports := []core.ContainerPort{{
		Name:          "gateway",
		ContainerPort: port,
		Protocol:      core.ProtocolTCP,
	}}

	probeHandler := core.ProbeHandler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port,
			},
		},
	}

	livenessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 10,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	//env := o.(*crd.ClowdEnvironment)
	//image := provutils.GetCaddyImage(env)
	image := "127.0.0.1:5000/caddy:02"

	c := core.Container{
		Name:           nn.Name,
		Image:          image,
		Ports:          ports,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		Resources: core.ResourceRequirements{
			Limits: core.ResourceList{
				"memory": resource.MustParse("750Mi"),
				"cpu":    resource.MustParse("1"),
			},
			Requests: core.ResourceList{
				"memory": resource.MustParse("400Mi"),
				"cpu":    resource.MustParse("100m"),
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
		VolumeMounts: []core.VolumeMount{
			{
				Name:      "config",
				MountPath: "/etc/caddy",
			},
			{
				Name:      "certs",
				MountPath: "/certs",
			},
			{
				Name:      "ca",
				MountPath: "/cas",
			},
		},
	}

	dd.Spec.Template.Spec.Volumes = []core.Volume{
		{
			Name: "config",
			VolumeSource: core.VolumeSource{
				ConfigMap: &core.ConfigMapVolumeSource{
					LocalObjectReference: core.LocalObjectReference{
						Name: providers.GetNamespacedName(o, "caddy-gateway").Name,
					},
				},
			},
		},
		{
			Name: "certs",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: providers.GetNamespacedName(o, "caddy-gateway").Name,
				},
			},
		},
		{
			Name: "ca",
			VolumeSource: core.VolumeSource{
				ConfigMap: &core.ConfigMapVolumeSource{
					LocalObjectReference: core.LocalObjectReference{
						Name: "cacert",
					},
				},
			},
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:       "gateway",
		Port:       port,
		Protocol:   "TCP",
		TargetPort: intstr.FromInt(int(port)),
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, false)
}
