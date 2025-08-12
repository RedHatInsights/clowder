package web

import (
	"crypto/sha256"
	"fmt"
	"sort"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

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
)

// WebGatewayDeployment is the resource ident for the web gateway deployment
var WebGatewayDeployment = rc.NewSingleResourceIdent(ProvName, "web_gateway_deployment", &apps.Deployment{})

// WebGatewayIngress is the resource ident for the web gateway ingress
var WebGatewayIngress = rc.NewSingleResourceIdent(ProvName, "web_gateway_ingress", &networking.Ingress{})

// WebGatewayService is the resource ident for the web gateway service
var WebGatewayService = rc.NewSingleResourceIdent(ProvName, "web_gateway_service", &core.Service{})

// WebGatewayConfigMap is the resource ident for the web gateway config map
var WebGatewayConfigMap = rc.NewSingleResourceIdent(ProvName, "web_gateway_configmap", &core.Service{})

// WebGatewayCertificateIssuer is the resource ident for the web gateway certificate issuer
var WebGatewayCertificateIssuer = rc.NewSingleResourceIdent(ProvName, "web_gateway_cert_issuer", &certmanager.Issuer{})

// WebGatewayCertificate is the resource ident for the web gateway certificate
var WebGatewayCertificate = rc.NewSingleResourceIdent(ProvName, "web_gateway_certificate", &certmanager.Certificate{})

func configureWebGateway(web *localWebProvider) error {
	if !web.Env.Spec.Providers.Web.GatewayCert.Enabled {
		return nil
	}

	if err := makeWebGatewayIngress(&web.Provider); err != nil {
		return err
	}

	configHashCaddy := ""

	configHashCaddy, err := makeWebGatewayConfigMap(&web.Provider)
	if err != nil {
		return err
	}

	if err := makeWebGatewayCertificateIssuer(&web.Provider); err != nil {
		return err
	}

	if err := makeWebGatewayCertificate(&web.Provider); err != nil {
		return err
	}

	objList := []rc.ResourceIdent{
		WebGatewayDeployment,
		WebGatewayService,
	}

	if err := providers.CachedMakeComponent(web, objList, web.Env, "caddy-gateway", makeWebGatewayDeployment, false); err != nil {
		return err
	}

	webDep := &apps.Deployment{}
	if err := web.Cache.Get(WebGatewayDeployment, webDep); err != nil {
		return err
	}

	annotations := map[string]string{
		"clowder/authsidecar-confighash": configHashCaddy,
	}

	if web.Env.Spec.Providers.Web.GatewayCert.LocalCAConfigMap != "" {
		webDep.Spec.Template.Spec.Volumes = append(webDep.Spec.Template.Spec.Volumes,
			core.Volume{
				Name: "ca",
				VolumeSource: core.VolumeSource{
					ConfigMap: &core.ConfigMapVolumeSource{
						LocalObjectReference: core.LocalObjectReference{
							Name: web.Env.Spec.Providers.Web.GatewayCert.LocalCAConfigMap,
						},
					},
				},
			},
		)
		webDep.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			webDep.Spec.Template.Spec.Containers[0].VolumeMounts, core.VolumeMount{
				Name:      "ca",
				MountPath: "/cas",
			},
		)
	}

	webDep.Spec.Template.Spec.Containers[0].Image = provutils.GetCaddyGatewayImage(web.Env)

	utils.UpdateAnnotations(&webDep.Spec.Template, annotations)

	return web.Cache.Update(WebGatewayDeployment, webDep)
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

	if p.Env.Spec.Providers.Web.GatewayCert.CertMode == "acme" {
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

	if p.Env.Spec.Providers.Web.GatewayCert.CertMode == "acme" {
		cert, err := acmeIssuerSpec(p)
		if err != nil {
			return err
		}
		certi.Spec = *cert
	} else {
		certi.Spec = *selfSignedIssuerSpec()
	}

	return p.Cache.Update(WebGatewayCertificateIssuer, certi)
}

func acmeIssuerSpec(p *providers.Provider) (*certmanager.IssuerSpec, error) {
	if p.Env.Spec.Providers.Web.GatewayCert.EmailAddress == "" {
		return nil, fmt.Errorf("could not get env.Spec.EmailAddress for Cert")
	}
	return &certmanager.IssuerSpec{
		IssuerConfig: certmanager.IssuerConfig{
			ACME: &acme.ACMEIssuer{
				Email:          p.Env.Spec.Providers.Web.GatewayCert.EmailAddress,
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
	}, nil
}

func selfSignedIssuerSpec() *certmanager.IssuerSpec {
	return &certmanager.IssuerSpec{
		IssuerConfig: certmanager.IssuerConfig{
			SelfSigned: &certmanager.SelfSignedIssuer{},
		},
	}
}

func acmeCert(p *providers.Provider) *certmanager.CertificateSpec {
	return &certmanager.CertificateSpec{
		DNSNames: []string{
			getCertHostname(p.Env.Status.Hostname),
		},
		IssuerRef: v1.ObjectReference{
			Group: "cert-manager.io",
			Kind:  "Issuer",
			Name:  p.Env.GetClowdNamespace(),
		},
		SecretName: providers.GetNamespacedName(p.Env, "caddy-gateway").Name,
	}
}

func selfSignedCert(p *providers.Provider) *certmanager.CertificateSpec {
	return &certmanager.CertificateSpec{
		CommonName: getCertHostname(p.Env.Status.Hostname),
		DNSNames: []string{
			getCertHostname(p.Env.Status.Hostname),
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

func makeWebGatewayConfigMap(p *providers.Provider) (string, error) {
	cm := &core.ConfigMap{}
	snn := providers.GetNamespacedName(p.Env, "caddy-gateway")

	if err := p.Cache.Create(CoreCaddyConfigMap, snn, cm); err != nil {
		return "", err
	}

	cm.Name = snn.Name
	cm.Namespace = snn.Namespace
	cm.OwnerReferences = []metav1.OwnerReference{p.Env.MakeOwnerReference()}

	appList, err := p.Env.GetAppsInEnv(p.Ctx, p.Client)
	if err != nil {
		return "", err
	}
	bopHostname := fmt.Sprintf("%s-%s.%s.svc:8090", p.Env.GetClowdName(), "mbop", p.Env.GetClowdNamespace())

	upstreamList, whitelistStrings := buildUpstreamAndWhiteLists(bopHostname, appList)

	cmData, err := GenerateConfig(
		getCertHostname(p.Env.Status.Hostname),
		fmt.Sprintf("http://%s", bopHostname),
		whitelistStrings,
		upstreamList,
	)
	if err != nil {
		return "", err
	}

	cm.Data = map[string]string{
		"Caddyfile.json": cmData,
	}

	h := sha256.New()
	h.Write([]byte(cmData))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	return hash, p.Cache.Update(CoreCaddyConfigMap, cm)
}

func buildUpstreamAndWhiteLists(bopHostname string, appList *crd.ClowdAppList) ([]ProxyRoute, []string) {
	whitelistStrings := []string{}
	upstreamList := []ProxyRoute{{
		Upstream: bopHostname,
		Path:     "/v1/registrations*",
	}, {
		Upstream: bopHostname,
		Path:     "/v1/check_registration*",
	}}

	appMap := map[string]crd.ClowdApp{}
	names := []string{}

	for _, app := range appList.Items {
		name := fmt.Sprintf("%s-%s", app.Name, app.Namespace)
		names = append(names, name)
		appMap[name] = app
	}

	sort.Strings(names)

	for _, name := range names {
		innerApp := appMap[name]
		for _, deployment := range innerApp.Spec.Deployments {
			innerDeployment := deployment
			whitelistStrings = append(whitelistStrings, innerDeployment.WebServices.Public.WhitelistPaths...)

			if !innerDeployment.WebServices.Public.Enabled && !bool(innerDeployment.Web) {
				continue
			}

			name := innerApp.GetDeploymentNamespacedName(&innerDeployment).Name
			hostname := fmt.Sprintf("%s.%s.svc", name, innerApp.Namespace)

			if innerDeployment.WebServices.Public.APIPaths != nil {
				// apiPaths was defined, use it and ignore 'apiPath'
				for _, apiPath := range innerDeployment.WebServices.Public.APIPaths {
					upstreamList = append(upstreamList, ProxyRoute{
						Upstream: fmt.Sprintf("%s:%d", hostname, 8000),
						Path:     string(apiPath),
					})
				}
			} else {
				apiPath := innerDeployment.WebServices.Public.APIPath

				if apiPath == "" {
					apiPath = innerDeployment.Name
				}

				upstreamList = append(upstreamList, ProxyRoute{
					Upstream: fmt.Sprintf("%s:%d", hostname, 8000),
					Path:     fmt.Sprintf("/api/%s/*", apiPath),
				})

			}
		}
	}

	return upstreamList, whitelistStrings
}

func makeWebGatewayDeployment(_ *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, _ bool) error {
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

	dd.Spec.Template.Labels = labels

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
	command := []string{"caddy", "run", "--config", "/etc/caddy/Caddyfile.json"}
	c := core.Container{
		Name:           nn.Name,
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
		},
		Command: command,
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
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:     "gateway",
		Port:     port,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, false)
	return nil
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

	ingressClass := p.Env.Spec.Providers.Web.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	path := networking.HTTPIngressPath{
		Backend: networking.IngressBackend{
			Service: &networking.IngressServiceBackend{
				Name: nn.Name,
				Port: networking.ServiceBackendPort{
					Name: "gateway",
				},
			},
		},
	}

	if ingressClass == "nginx" {
		utils.UpdateAnnotations(netobj, map[string]string{
			"nginx.ingress.kubernetes.io/ssl-passthrough":  "true",
			"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
		})
		path.Path = "/"
		path.PathType = (*networking.PathType)(utils.StringPtr("Prefix"))
	} else {
		utils.UpdateAnnotations(netobj, map[string]string{
			"route.openshift.io/termination": "passthrough",
		})
		path.Path = ""
		path.PathType = (*networking.PathType)(utils.StringPtr("ImplementationSpecific"))
	}

	netobj.Spec = networking.IngressSpec{
		IngressClassName: &ingressClass,
		Rules: []networking.IngressRule{
			{
				Host: getCertHostname(p.Env.Status.Hostname),
				IngressRuleValue: networking.IngressRuleValue{
					HTTP: &networking.HTTPIngressRuleValue{
						Paths: []networking.HTTPIngressPath{path},
					},
				},
			},
		},
	}

	return p.Cache.Update(WebGatewayIngress, netobj)
}
