package web

import (
	"fmt"
	"os"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

var KEYCLOAK_VERSION = "15.0.2"
var MBOP_IMAGE = "quay.io/cloudservices/mbop:0d3f99f"
var MOCKTITLEMENTS_IMAGE = "quay.io/cloudservices/mocktitlements:130433d"

func makeService(cache *rc.ObjectCache, deployment *crd.Deployment, app *crd.ClowdApp, env *crd.ClowdEnvironment) error {

	s := &core.Service{}
	nn := app.GetDeploymentNamespacedName(deployment)

	if err := cache.Create(CoreService, nn, s); err != nil {
		return err
	}

	d := &apps.Deployment{}

	cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(deployment))

	servicePorts := []core.ServicePort{}
	containerPorts := []core.ContainerPort{}

	if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
		appProtocol := "http"
		// Create the core service port
		webPort := core.ServicePort{
			Name:        "public",
			Port:        env.Spec.Providers.Web.Port,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
			TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.Port)),
		}

		servicePorts = append(servicePorts, webPort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "web",
				ContainerPort: env.Spec.Providers.Web.Port,
				Protocol:      core.ProtocolTCP,
			},
		)

		if env.Spec.Providers.Web.Mode == "local" {
			authPortNumber := env.Spec.Providers.Web.AuthPort

			if authPortNumber == 0 {
				authPortNumber = 8080
			}
			authPort := core.ServicePort{
				Name:        "auth",
				Port:        authPortNumber,
				Protocol:    "TCP",
				AppProtocol: &appProtocol,
				TargetPort:  intstr.FromInt(int(authPortNumber)),
			}
			servicePorts = append(servicePorts, authPort)
		}
	}

	if deployment.WebServices.Private.Enabled {
		privatePort := env.Spec.Providers.Web.PrivatePort

		appProtocol := "http"
		if deployment.WebServices.Private.AppProtocol != "" {
			appProtocol = string(deployment.WebServices.Private.AppProtocol)
		}

		if privatePort == 0 {
			privatePort = 10000
		}

		webPort := core.ServicePort{
			Name:        "private",
			Port:        privatePort,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
			TargetPort:  intstr.FromInt(int(privatePort)),
		}
		servicePorts = append(servicePorts, webPort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "private",
				ContainerPort: privatePort,
				Protocol:      core.ProtocolTCP,
			},
		)
	}

	utils.MakeService(s, nn, map[string]string{"pod": nn.Name}, servicePorts, app, env.IsNodePort())

	d.Spec.Template.Spec.Containers[0].Ports = containerPorts

	if err := cache.Update(CoreService, s); err != nil {
		return err
	}

	if err := cache.Update(deployProvider.CoreDeployment, d); err != nil {
		return err
	}

	return nil
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

	if err := cache.Update(WebKeycloakImportSecret, userData); err != nil {
		return err
	}

	return nil
}

func makeKeycloak(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
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
			Name:  "DB_VENDOR",
			Value: "h2",
		},
		{
			Name:  "PROXY_ADDRESS_FORWARDING",
			Value: "true",
		},
		{
			Name: "KEYCLOAK_USER",
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
			Name: "KEYCLOAK_PASSWORD",
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

	image := fmt.Sprintf("quay.io/keycloak/keycloak:%s", KEYCLOAK_VERSION)

	if clowderconfig.LoadedConfig.Images.Keycloak != "" {
		image = clowderconfig.LoadedConfig.Images.Keycloak
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          image,
		Env:            envVars,
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
				MountPath: "/json",
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

func makeBOP(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
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

	dd.Spec.Template.ObjectMeta.Labels = labels

	envVars := []core.EnvVar{
		{
			Name:  "KEYCLOAK_SERVER",
			Value: fmt.Sprintf("http://%s-keycloak.%s.svc:8080", o.GetClowdName(), o.GetClowdNamespace()),
		},
		{
			Name: "KEYCLOAK_USERNAME",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "username",
				},
			},
		},
		{
			Name: "KEYCLOAK_PASSWORD",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "password",
				},
			},
		},
		{
			Name: "KEYCLOAK_VERSION",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "version",
				},
			},
		},
	}

	port := int32(8090)

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

	image := MBOP_IMAGE

	if clowderconfig.LoadedConfig.Images.MBOP != "" {
		image = clowderconfig.LoadedConfig.Images.MBOP
	}

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
	}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)

}

func makeMocktitlements(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
	snn := providers.GetNamespacedName(o, "keycloak")
	nn := providers.GetNamespacedName(o, "mocktitlements")

	dd := objMap[WebMocktitlementsDeployment].(*apps.Deployment)
	svc := objMap[WebMocktitlementsService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	annotations := map[string]string{
		"clowder/authsidecar-image":   "a76bb81",
		"clowder/authsidecar-enabled": "true",
		"clowder/authsidecar-port":    "8090",
		"clowder/authsidecar-config":  "caddy-config-mocktitlements",
	}

	utils.UpdatePodTemplateAnnotations(&dd.Spec.Template, annotations)

	envVars := []core.EnvVar{
		{
			Name:  "KEYCLOAK_SERVER",
			Value: fmt.Sprintf("http://%s-keycloak.%s.svc:8080", o.GetClowdName(), o.GetClowdNamespace()),
		},
		{
			Name: "KEYCLOAK_USERNAME",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "username",
				},
			},
		},
		{
			Name: "KEYCLOAK_PASSWORD",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "password",
				},
			},
		},
		{
			Name: "KEYCLOAK_VERSION",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "version",
				},
			},
		},
	}

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

	image := MOCKTITLEMENTS_IMAGE

	if clowderconfig.LoadedConfig.Images.MBOP != "" {
		image = clowderconfig.LoadedConfig.Images.MBOP
	}

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
			Name:       "mocktitlements",
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

}
