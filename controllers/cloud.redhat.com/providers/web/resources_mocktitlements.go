package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// WebBOPDeployment is the mocked bop deployment
var WebMocktitlementsDeployment = rc.NewSingleResourceIdent(ProvName, "web_mocktitlements_deployment", &apps.Deployment{})

// WebKeycloakService is the mocked keycloak deployment
var WebMocktitlementsService = rc.NewSingleResourceIdent(ProvName, "web_mocktitlements_service", &core.Service{})

// WebKeycloakIngress is the mocked bop ingress
var WebMocktitlementsIngress = rc.NewSingleResourceIdent(ProvName, "web_mocktitlements_ingress", &networking.Ingress{})

func configureMocktitlements(web *localWebProvider) error {

	objList := []rc.ResourceIdent{
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
