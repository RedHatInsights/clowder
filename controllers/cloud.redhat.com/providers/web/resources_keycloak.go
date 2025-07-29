package web

import (
	"fmt"
	"os"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"
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

// WebKeycloakDBDeployment is the ident referring to the local Feature Flags DB deployment object.
var WebKeycloakDBDeployment = rc.NewSingleResourceIdent(ProvName, "web_keycloak_db_deployment", &apps.Deployment{})

// WebKeycloakDBService is the ident referring to the local Feature Flags DB service object.
var WebKeycloakDBService = rc.NewSingleResourceIdent(ProvName, "web_keycloak_db_service", &core.Service{})

// WebKeycloakDBPVC is the ident referring to the local Feature Flags DB PVC object.
var WebKeycloakDBPVC = rc.NewSingleResourceIdent(ProvName, "web_keycloak_db_pvc", &core.PersistentVolumeClaim{})

// WebKeycloakDBSecret is the ident referring to the local Feature Flags DB secret object.
var WebKeycloakDBSecret = rc.NewSingleResourceIdent(ProvName, "web_keycloak_db_secret", &core.Secret{})

func configureKeycloakDB(web *localWebProvider) error {
	namespacedNameDb := types.NamespacedName{
		Name:      "keycloak-db",
		Namespace: web.Env.Status.TargetNamespace,
	}

	dd := &apps.Deployment{}
	if err := web.Cache.Create(WebKeycloakDBDeployment, namespacedNameDb, dd); err != nil {
		return err
	}

	dbCfg := config.DatabaseConfig{}

	password, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return errors.Wrap("password generate failed", err)
	}

	pgPassword, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return errors.Wrap("pgPassword generate failed", err)
	}

	username := utils.RandString(16)
	hostname := fmt.Sprintf("%v.%v.svc", namespacedNameDb.Name, namespacedNameDb.Namespace)

	dataInitDb := func() map[string]string {

		return map[string]string{
			"hostname": hostname,
			"port":     "5432",
			"username": username,
			"password": password,
			"pgPass":   pgPassword,
			"name":     "keycloak",
		}
	}

	secMapDb, err := providers.MakeOrGetSecret(web.Env, web.Cache, WebKeycloakDBSecret, namespacedNameDb, dataInitDb)
	if err != nil {
		return errors.Wrap("Couldn't set/get secret", err)
	}

	err = dbCfg.Populate(secMapDb)
	if err != nil {
		return errors.Wrap("couldn't convert to int", err)
	}
	dbCfg.AdminUsername = "postgres"

	labels := &map[string]string{"sub": "keycloak"}

	res := core.ResourceRequirements{
		Limits: core.ResourceList{
			"memory": resource.MustParse("200Mi"),
			"cpu":    resource.MustParse("100m"),
		},
		Requests: core.ResourceList{
			"memory": resource.MustParse("100Mi"),
			"cpu":    resource.MustParse("50m"),
		},
	}

	dbImage, err := provutils.GetDefaultDatabaseImage(15)
	if err != nil {
		return err
	}

	provutils.MakeLocalDB(dd, namespacedNameDb, web.Env, labels, &dbCfg, dbImage, web.Env.Spec.Providers.Web.KeycloakPVC, "keycloak", &res)

	if err = web.Cache.Update(WebKeycloakDBDeployment, dd); err != nil {
		return err
	}

	s := &core.Service{}
	if err := web.Cache.Create(WebKeycloakDBService, namespacedNameDb, s); err != nil {
		return err
	}

	provutils.MakeLocalDBService(s, namespacedNameDb, web.Env, labels)

	if err = web.Cache.Update(WebKeycloakDBService, s); err != nil {
		return err
	}

	if web.Env.Spec.Providers.Web.KeycloakPVC {
		pvc := &core.PersistentVolumeClaim{}
		if err = web.Cache.Create(WebKeycloakDBPVC, namespacedNameDb, pvc); err != nil {
			return err
		}

		provutils.MakeLocalDBPVC(pvc, namespacedNameDb, web.Env, sizing.GetDefaultVolCapacity())

		if err = web.Cache.Update(WebKeycloakDBPVC, pvc); err != nil {
			return err
		}
	}

	return nil
}

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

	if err := providers.CachedMakeComponent(web, objList, web.Env, "keycloak", makeKeycloak, false); err != nil {
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

func baseProbeHandler(port int32, path string) core.ProbeHandler {
	return core.ProbeHandler{
		HTTPGet: &core.HTTPGetAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port,
			},
			Scheme: core.URISchemeHTTP,
			HTTPHeaders: []core.HTTPHeader{
				{
					Name:  "Accept",
					Value: "application/json",
				},
			},
			Path: path,
		},
	}
}

func makeKeycloak(_ *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) error {
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

	dd.Spec.Template.Labels = labels

	envVars := []core.EnvVar{
		{
			Name:  "KC_DB",
			Value: "postgres",
		},
		{
			Name:  "KC_DB_URL_PORT",
			Value: "5432",
		},
		{
			Name:  "PROXY_ADDRESS_FORWARDING",
			Value: "true",
		},
		{
			Name:  "KEYCLOAK_IMPORT",
			Value: "/json/redhat-external-realm.json",
		},
	}

	envVars = provutils.AppendEnvVarsFromSecret(envVars, "keycloak-db",
		provutils.NewSecretEnvVar("KC_DB_USERNAME", "username"),
		provutils.NewSecretEnvVar("KC_DB_PASSWORD", "password"),
		provutils.NewSecretEnvVar("KC_DB_URL_DATABASE", "name"),
		provutils.NewSecretEnvVar("KC_DB_URL_HOST", "hostname"),
	)

	envVars = provutils.AppendEnvVarsFromSecret(envVars, nn.Name,
		provutils.NewSecretEnvVar("KEYCLOAK_ADMIN", "username"),
		provutils.NewSecretEnvVar("KEYCLOAK_ADMIN_PASSWORD", "password"),
	)

	port := int32(8080)

	ports := []core.ContainerPort{{
		Name:          "service",
		ContainerPort: port,
		Protocol:      core.ProtocolTCP,
	}}

	livenessProbe := core.Probe{
		ProbeHandler:        baseProbeHandler(port, "auth/health/live"),
		InitialDelaySeconds: 60,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        baseProbeHandler(port, "auth/health/ready"),
		InitialDelaySeconds: 60,
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
			"start",
			"--import-realm",
			"--hostname-strict",
			"false",
			"--http-enabled",
			"true",
			"--http-relative-path",
			"/auth",
			"--health-enabled",
			"true",
			"--metrics-enabled",
			"true",
			"--proxy",
			"edge",
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
						Paths: []networking.HTTPIngressPath{
							{
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
							},
							{
								Path:     "/auth/realms/redhat-external/apis/service_accounts/v1",
								PathType: (*networking.PathType)(utils.StringPtr("Prefix")),
								Backend: networking.IngressBackend{
									Service: &networking.IngressServiceBackend{
										Name: fmt.Sprintf("%s-mocktitlements", p.Env.Name),
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

	return p.Cache.Update(WebKeycloakIngress, netobj)
}
