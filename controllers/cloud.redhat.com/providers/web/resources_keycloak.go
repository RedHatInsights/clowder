package web

import (
	"fmt"
	"os"
	"strings"

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

// WebKeycloakDeployment is the mocked keycloak deployment
var WebKeycloakDeployment = rc.NewSingleResourceIdent(ProvName, "web_keycloak_deployment", &apps.Deployment{}, rc.ResourceOptions{WriteNow: true})

// WebKeycloakService is the mocked keycloak deployment
var WebKeycloakService = rc.NewSingleResourceIdent(ProvName, "web_keycloak_service", &core.Service{}, rc.ResourceOptions{WriteNow: true})

// WebKeycloakIngress is the keycloak ingress
var WebKeycloakIngress = rc.NewSingleResourceIdent(ProvName, "web_keycloak_ingress", &networking.Ingress{})

// WebKeycloakImportSecret is the keycloak import secret
var WebKeycloakImportSecret = rc.NewSingleResourceIdent(ProvName, "web_keycloak_import_secret", &core.Secret{})

// WebKeycloakSecret is the mocked secret config
var WebKeycloakSecret = rc.NewSingleResourceIdent(ProvName, "web_keycloak_secret", &core.Secret{}, rc.ResourceOptions{WriteNow: true})

func configureKeycloak(web *localWebProvider) error {
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

	return makeAuthIngress(&web.Provider)
}

func makeKeycloakImportSecretRealm(cache *rc.ObjectCache, o obj.ClowdObject, password string) error {
	userData := &core.Secret{}
	userDataNN := providers.GetNamespacedName(o, "keycloak-realm-import")

	if err := cache.Create(WebKeycloakImportSecret, userDataNN, userData); err != nil {
		return err
	}

	labels := o.GetLabels()
	labels["env-app"] = userDataNN.Name

	labeler := utils.MakeLabeler(userDataNN, labels, o)

	labeler(userData)

	userImportData, err := os.ReadFile("./jsons/redhat-external-realm.json")
	if err != nil {
		return fmt.Errorf("could not read user data: %w", err)
	}

	userData.StringData = map[string]string{}
	userImportDataString := string(userImportData)
	userImportDataString = strings.Replace(userImportDataString, "########PASSWORD########", password, 1)

	userData.StringData["redhat-external-realm.json"] = string(userImportDataString)

	return cache.Update(WebKeycloakImportSecret, userData)
}

func makeKeycloak(o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) {
	nn := providers.GetNamespacedName(o, "keycloak")

	dd := objMap[WebKeycloakDeployment].(*apps.Deployment)
	svc := objMap[WebKeycloakService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	envVars := []core.EnvVar{
		{
			Name:  "KC_DB",
			Value: "dev-mem",
		},
		{
			Name:  "PROXY_ADDRESS_FORWARDING",
			Value: "true",
		},
		{
			Name: "KEYCLOAK_ADMIN",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: nn.Name,
					},
					Key: "username",
				},
			},
		},
		{
			Name: "KEYCLOAK_ADMIN_PASSWORD",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: nn.Name,
					},
					Key: "password",
				},
			},
		},
		{
			Name:  "KEYCLOAK_IMPORT",
			Value: "/json/redhat-external-realm.json",
		},
	}

	port := int32(8080)

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
		InitialDelaySeconds: 30,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 30,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	env := o.(*crd.ClowdEnvironment)
	image := provutils.GetKeycloakImage(env)

	c := core.Container{
		Name:  nn.Name,
		Image: image,
		Env:   envVars,
		Args: []string{
			"start-dev",
			"--import-realm",
		},
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
				Name:      "realm-import",
				MountPath: "/opt/keycloak/data/import/",
			},
		},
	}

	dd.Spec.Template.Spec.Volumes = []core.Volume{
		{
			Name: "realm-import",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: providers.GetNamespacedName(o, "keycloak-realm-import").Name,
				},
			},
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:       "keycloak",
		Port:       port,
		Protocol:   "TCP",
		TargetPort: intstr.FromInt(int(port)),
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)

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
