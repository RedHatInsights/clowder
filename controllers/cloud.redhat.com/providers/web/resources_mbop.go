package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// WebBOPDeployment is the mocked bop deployment
var WebBOPDeployment = rc.NewSingleResourceIdent(ProvName, "web_bop_deployment", &apps.Deployment{})

// WebKeycloakService is the mocked keycloak deployment
var WebBOPService = rc.NewSingleResourceIdent(ProvName, "web_bop_service", &core.Service{})

// WebKeycloakIngress is the mocked bop ingress
var WebBOPIngress = rc.NewSingleResourceIdent(ProvName, "web_bop_ingress", &networking.Ingress{})

func configureMBOP(web *localWebProvider) error {

	objList := []rc.ResourceIdent{
		WebBOPDeployment,
		WebBOPService,
	}

	if err := providers.CachedMakeComponent(web, objList, web.Env, "mbop", makeBOP, false); err != nil {
		return err
	}

	if err := makeMBOPSecret(&web.Provider); err != nil {
		return err
	}

	return makeBOPIngress(&web.Provider)
}

func makeMBOPSecret(p *providers.Provider) error {
	nn := types.NamespacedName{
		Name:      "caddy-config-mbop",
		Namespace: p.Env.GetClowdNamespace(),
	}

	sec := &core.Secret{}
	if err := p.Cache.Create(WebSecret, nn, sec); err != nil {
		return err
	}

	sec.Name = nn.Name
	sec.Namespace = nn.Namespace
	sec.OwnerReferences = []metav1.OwnerReference{p.Env.MakeOwnerReference()}
	sec.Type = core.SecretTypeOpaque

	envSec := &core.Secret{}
	envSecnn := providers.GetNamespacedName(p.Env, "keycloak")
	if err := p.Client.Get(p.Ctx, envSecnn, envSec); err != nil {
		return err
	}

	sec.StringData = map[string]string{
		"bopurl":      "http://127.0.0.1:8090",
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
	if err := p.Cache.Get(WebBOPDeployment, d, dnn); err != nil {
		return err
	}

	annotations := map[string]string{
		"clowder/authsidecar-confighash": hash,
	}

	utils.UpdateAnnotations(&d.Spec.Template, annotations)

	if err := p.Cache.Update(WebBOPDeployment, d); err != nil {
		return err
	}

	return p.Cache.Update(WebSecret, sec)
}

func makeBOPIngress(p *providers.Provider) error {
	netobj := &networking.Ingress{}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-mbop", p.Env.Name),
		Namespace: p.Env.Status.TargetNamespace,
	}

	if err := p.Cache.Create(WebBOPIngress, nn, netobj); err != nil {
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
							Path:     "/v1/registrations",
							PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
							Backend: networking.IngressBackend{
								Service: &networking.IngressServiceBackend{
									Name: fmt.Sprintf("%s-mbop", p.Env.Name),
									Port: networking.ServiceBackendPort{
										Name: "auth",
									},
								},
							},
						},
							{
								Path:     "/v1/check_registration",
								PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
								Backend: networking.IngressBackend{
									Service: &networking.IngressServiceBackend{
										Name: fmt.Sprintf("%s-mbop", p.Env.Name),
										Port: networking.ServiceBackendPort{
											Name: "auth",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return p.Cache.Update(WebBOPIngress, netobj)
}

func makeBOP(_ *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) error {
	snn := providers.GetNamespacedName(o, "keycloak")
	nn := providers.GetNamespacedName(o, "mbop")

	dd := objMap[WebBOPDeployment].(*apps.Deployment)
	svc := objMap[WebBOPService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.Labels = labels

	env := o.(*crd.ClowdEnvironment)
	caddyImage := provutils.GetCaddyImage(env)

	annotations := map[string]string{
		"clowder/authsidecar-image":   caddyImage,
		"clowder/authsidecar-enabled": "true",
		"clowder/authsidecar-port":    "8090",
		"clowder/authsidecar-config":  "caddy-config-mbop",
	}

	utils.UpdateAnnotations(&dd.Spec.Template, annotations)

	envVars := []core.EnvVar{
		{
			Name:  "KEYCLOAK_SERVER",
			Value: fmt.Sprintf("http://%s-keycloak.%s.svc:8080", o.GetClowdName(), o.GetClowdNamespace()),
		},
	}

	envVars = provutils.AppendEnvVarsFromSecret(envVars, snn.Name,
		provutils.NewSecretEnvVar("KEYCLOAK_USERNAME", "username"),
		provutils.NewSecretEnvVar("KEYCLOAK_PASSWORD", "password"),
		provutils.NewSecretEnvVar("KEYCLOAK_VERSION", "version"),
	)

	port := int32(8090)
	authPort := int32(8080)

	ports := []core.ContainerPort{{
		Name:          "service",
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
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      2,
	}

	image := provutils.GetMockBOPImage(env)

	c := core.Container{
		Name:           nn.Name,
		Image:          image,
		Env:            envVars,
		Ports:          ports,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		Resources: core.ResourceRequirements{
			Limits: core.ResourceList{
				"memory": resource.MustParse("200Mi"),
				"cpu":    resource.MustParse("100m"),
			},
			Requests: core.ResourceList{
				"memory": resource.MustParse("100Mi"),
				"cpu":    resource.MustParse("50m"),
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{
		{
			Name:       "mbop",
			Port:       port,
			Protocol:   "TCP",
			TargetPort: intstr.FromInt(int(port)),
		},
		{
			Name:       "auth",
			Port:       authPort,
			Protocol:   "TCP",
			TargetPort: intstr.FromInt(int(authPort)),
		},
	}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)
	return nil
}
