package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provDeploy "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// WebSecret is the mocked secret config
var WebSecret = rc.NewMultiResourceIdent(ProvName, "web_secret", &core.Secret{})

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
		WebKeycloakDBDeployment,
		WebKeycloakDBPVC,
		WebKeycloakDBService,
		WebKeycloakDBSecret,
		WebBOPDeployment,
		WebBOPService,
		WebMocktitlementsDeployment,
		WebMocktitlementsService,
		WebMocktitlementsIngress,
		WebSecret,
		WebIngress,
		WebKeycloakSecret,
		WebGatewayDeployment,
		WebGatewayIngress,
		WebGatewayService,
		WebGatewayConfigMap,
		WebGatewayCertificate,
		WebGatewayCertificateIssuer,
		CoreCaddyConfigMap,
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

	if err := configureKeycloakDB(web); err != nil {
		return err
	}

	if err := configureKeycloak(web); err != nil {
		return err
	}

	if err := configureMBOP(web); err != nil {
		return err
	}

	if err := configureMocktitlements(web); err != nil {
		return err
	}

	return configureWebGateway(web)
}

func (web *localWebProvider) Provide(app *crd.ClowdApp) error {

	web.Config.WebPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	web.Config.PublicPort = utils.IntPtr(int(web.Env.Spec.Providers.Web.Port))
	web.Config.Hostname = &web.Env.Status.Hostname

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

		if err := web.createIngress(app, &innerDeployment); err != nil {
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

	apiPaths := provutils.GetAPIPaths(deployment, nn.Name)

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
						Paths: []networking.HTTPIngressPath{},
					},
				},
			},
		},
	}

	for _, path := range apiPaths {
		path := networking.HTTPIngressPath{
			Path:     path,
			PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
			Backend: networking.IngressBackend{
				Service: &networking.IngressServiceBackend{
					Name: nn.Name,
					Port: networking.ServiceBackendPort{
						Name: "auth",
					},
				},
			},
		}
		netobj.Spec.Rules[0].IngressRuleValue.HTTP.Paths = append(netobj.Spec.Rules[0].IngressRuleValue.HTTP.Paths, path)
	}

	return web.Cache.Update(WebIngress, netobj)
}

func (web *localWebProvider) populateCA() error {
	if web.Env.Spec.Providers.Web.TLS.Enabled {
		web.Config.TlsCAPath = utils.StringPtr("/cdapp/certs/service-ca.crt")
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

func getAuthHostname(hostname string) string {
	hostComponents := strings.Split(hostname, ".")
	hostComponents[0] += "-auth"
	return strings.Join(hostComponents, ".")
}

func getCertHostname(hostname string) string {
	hostComponents := strings.Split(hostname, ".")
	hostComponents[0] += "-cert"
	return strings.Join(hostComponents, ".")
}
