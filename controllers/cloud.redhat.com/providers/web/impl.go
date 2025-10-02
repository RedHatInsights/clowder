package web

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// CoreService is the service for the apps deployments.
var CoreService = rc.NewMultiResourceIdent(ProvName, "core_service", &core.Service{})

// CoreCaddyConfigMap represents the resource identifier for core Caddy configuration maps
var CoreCaddyConfigMap = rc.NewMultiResourceIdent(ProvName, "core_caddy_config_map", &core.ConfigMap{}, rc.ResourceOptions{WriteNow: true})

func isPublicTLSEnabled(deployment *crd.Deployment, env *crd.ClowdEnvironment) bool {
	if deployment.WebServices.Public.TLS != nil {
		return *deployment.WebServices.Public.TLS
	}
	return env.Spec.Providers.Web.TLS.Enabled
}

func isPrivateTLSEnabled(deployment *crd.Deployment, env *crd.ClowdEnvironment) bool {
	if deployment.WebServices.Private.TLS != nil {
		return *deployment.WebServices.Private.TLS
	}
	return env.Spec.Providers.Web.TLS.Enabled
}

func makeService(cache *rc.ObjectCache, deployment *crd.Deployment, app *crd.ClowdApp, env *crd.ClowdEnvironment) error {

	s := &core.Service{}
	nn := app.GetDeploymentNamespacedName(deployment)

	appProtocol := "http"

	if err := cache.Create(CoreService, nn, s); err != nil {
		return err
	}

	d := &apps.Deployment{}

	if err := cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(deployment)); err != nil {
		return err
	}

	servicePorts := []core.ServicePort{}
	containerPorts := []core.ContainerPort{}

	if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
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

		// Set session affinity if enabled
		if deployment.WebServices.Public.SessionAffinity {
			s.Spec.SessionAffinity = core.ServiceAffinityClientIP
		}

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
		appProtocolPriv := "http"
		if deployment.WebServices.Private.AppProtocol != "" {
			appProtocolPriv = string(deployment.WebServices.Private.AppProtocol)
		}

		if privatePort == 0 {
			privatePort = 10000
		}

		webPort := core.ServicePort{
			Name:        "private",
			Port:        privatePort,
			Protocol:    "TCP",
			AppProtocol: &appProtocolPriv,
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

	var pubTLS, privTLS bool
	var pubPort, privPort int32

	if isPublicTLSEnabled(deployment, env) && deployment.WebServices.Public.Enabled {
		tlsPort := core.ServicePort{
			Name:        "tls",
			Port:        env.Spec.Providers.Web.TLS.Port,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
			TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.TLS.Port)),
		}
		servicePorts = append(servicePorts, tlsPort)
		pubTLS = true
		pubPort = int32(env.Spec.Providers.Web.TLS.Port)
	}

	if isPrivateTLSEnabled(deployment, env) && deployment.WebServices.Private.Enabled {
		appProtocolPriv := "http"
		if deployment.WebServices.Private.AppProtocol != "" {
			appProtocolPriv = string(deployment.WebServices.Private.AppProtocol)
		}

		if appProtocolPriv == "http" {
			tlsPrivatePort := core.ServicePort{
				Name:        "tls-private",
				Port:        env.Spec.Providers.Web.TLS.PrivatePort,
				Protocol:    "TCP",
				AppProtocol: &appProtocolPriv,
				TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.TLS.PrivatePort)),
			}
			servicePorts = append(servicePorts, tlsPrivatePort)
			privTLS = true
			privPort = int32(env.Spec.Providers.Web.TLS.PrivatePort)
		}
	}

	if privTLS || pubTLS {
		if err := generateCaddyConfigMap(cache, nn, app, pubTLS, privTLS, pubPort, privPort, env); err != nil {
			return err
		}
		populateSideCar(d, nn.Name, env.Spec.Providers.Web.TLS.Port, env.Spec.Providers.Web.TLS.PrivatePort, pubTLS, privTLS, env)
		setServiceTLSAnnotations(s, nn.Name)
	}

	utils.MakeService(s, nn, map[string]string{"pod": nn.Name}, servicePorts, app, env.IsNodePort())

	d.Spec.Template.Spec.Containers[0].Ports = containerPorts

	if err := cache.Update(CoreService, s); err != nil {
		return err
	}

	return cache.Update(deployProvider.CoreDeployment, d)
}

func generateCaddyConfigMap(cache *rc.ObjectCache, nn types.NamespacedName, app *crd.ClowdApp, pub bool, priv bool, pubPort int32, privPort int32, env *crd.ClowdEnvironment) error {

	cm := &core.ConfigMap{}
	snn := types.NamespacedName{
		Name:      caddyConfigName(nn.Name),
		Namespace: nn.Namespace,
	}

	if err := cache.Create(CoreCaddyConfigMap, snn, cm); err != nil {
		return err
	}

	cm.Name = snn.Name
	cm.Namespace = snn.Namespace
	cm.OwnerReferences = []metav1.OwnerReference{app.MakeOwnerReference()}

	cmData, err := generateCaddyConfig(pub, priv, pubPort, privPort, env)
	if err != nil {
		return err
	}

	cm.Data = map[string]string{
		"caddy.json": cmData,
	}
	return cache.Update(CoreCaddyConfigMap, cm)
}

func populateSideCar(d *apps.Deployment, name string, port int32, privatePort int32, pub bool, priv bool, env *crd.ClowdEnvironment) {
	ports := []core.ContainerPort{}
	if pub {
		ports = append(ports, core.ContainerPort{
			Name:          "tls",
			ContainerPort: port,
			Protocol:      core.ProtocolTCP,
		})
	}
	if priv {
		ports = append(ports, core.ContainerPort{
			Name:          "tls-private",
			ContainerPort: privatePort,
			Protocol:      core.ProtocolTCP,
		})
	}

	container := core.Container{
		Name:    "caddy-tls",
		Image:   provutils.GetCaddyProxyImage(env),
		Command: []string{"/usr/bin/caddy"},
		Args: []string{
			"run", "--config", "/etc/caddy/caddy.json",
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      "caddy-tls",
				ReadOnly:  true,
				MountPath: "/certs",
			},
			{
				Name:      "caddy-config",
				ReadOnly:  true,
				MountPath: "/etc/caddy",
			},
		},
		Ports: ports,
	}
	caddyTLSVol := core.Volume{
		Name: "caddy-tls",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName: certSecretName(name),
			},
		},
	}
	caddyConfigVol := core.Volume{
		Name: "caddy-config",
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: caddyConfigName(d.Name),
				},
			},
		},
	}
	d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, container)
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, caddyConfigVol, caddyTLSVol)
}

func setServiceTLSAnnotations(s *core.Service, name string) {
	annos := map[string]string{
		"service.beta.openshift.io/serving-cert-secret-name": certSecretName(name),
	}
	utils.UpdateAnnotations(s, annos)
}

func certSecretName(name string) string {
	return fmt.Sprintf("%s-serving-cert", name)
}

func caddyConfigName(name string) string {
	return fmt.Sprintf("%s-caddy-config", name)
}
