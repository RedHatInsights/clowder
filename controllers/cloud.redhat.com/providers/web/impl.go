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

func makeService(cache *rc.ObjectCache, deployment *crd.Deployment, app *crd.ClowdApp, env *crd.ClowdEnvironment) error {

	s := &core.Service{}
	nn := app.GetDeploymentNamespacedName(deployment)

	appProtocol := "http"
	h2cAppProtocol := "h2c"

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

	// Add H2C public port if enabled
	if deployment.WebServices.Public.H2CEnabled && env.Spec.Providers.Web.H2CPort != 0 {
		h2cPort := core.ServicePort{
			Name:        "h2c",
			Port:        env.Spec.Providers.Web.H2CPort,
			Protocol:    "TCP",
			AppProtocol: &h2cAppProtocol,
			TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.H2CPort)),
		}
		servicePorts = append(servicePorts, h2cPort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "h2c",
				ContainerPort: env.Spec.Providers.Web.H2CPort,
				Protocol:      core.ProtocolTCP,
			},
		)
	}

	if deployment.WebServices.Private.Enabled {
		privatePort := env.Spec.Providers.Web.PrivatePort

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

	// Add H2C private port if enabled
	if deployment.WebServices.Private.H2CEnabled && env.Spec.Providers.Web.H2CPrivatePort != 0 {
		h2cPrivatePort := core.ServicePort{
			Name:        "h2c-private",
			Port:        env.Spec.Providers.Web.H2CPrivatePort,
			Protocol:    "TCP",
			AppProtocol: &h2cAppProtocol,
			TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.H2CPrivatePort)),
		}
		servicePorts = append(servicePorts, h2cPrivatePort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "h2c-private",
				ContainerPort: env.Spec.Providers.Web.H2CPrivatePort,
				Protocol:      core.ProtocolTCP,
			},
		)
	}

	var pubTLS, privTLS, pubH2CTLS, privH2CTLS bool
	var pubPort, privPort, pubH2CPort, privH2CPort int32

	if provutils.IsPublicTLSEnabled(&deployment.WebServices, &env.Spec.Providers.Web.TLS) && deployment.WebServices.Public.Enabled {
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

	if provutils.IsPrivateTLSEnabled(&deployment.WebServices, &env.Spec.Providers.Web.TLS) && deployment.WebServices.Private.Enabled {
		tlsPrivatePort := core.ServicePort{
			Name:        "tls-private",
			Port:        env.Spec.Providers.Web.TLS.PrivatePort,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
			TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.TLS.PrivatePort)),
		}
		servicePorts = append(servicePorts, tlsPrivatePort)
		privTLS = true
		privPort = int32(env.Spec.Providers.Web.TLS.PrivatePort)
	}

	// Add TLS H2C public port if enabled
	if provutils.IsPublicTLSEnabled(&deployment.WebServices, &env.Spec.Providers.Web.TLS) && deployment.WebServices.Public.H2CEnabled && env.Spec.Providers.Web.TLS.H2CPort != 0 {
		tlsH2CPort := core.ServicePort{
			Name:        "h2c-tls",
			Port:        env.Spec.Providers.Web.TLS.H2CPort,
			Protocol:    "TCP",
			AppProtocol: &h2cAppProtocol,
			TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.TLS.H2CPort)),
		}
		servicePorts = append(servicePorts, tlsH2CPort)
		pubH2CTLS = true
		pubH2CPort = int32(env.Spec.Providers.Web.TLS.H2CPort)
	}

	// Add TLS H2C private port if enabled
	if provutils.IsPrivateTLSEnabled(&deployment.WebServices, &env.Spec.Providers.Web.TLS) && deployment.WebServices.Private.H2CEnabled && env.Spec.Providers.Web.TLS.H2CPrivatePort != 0 {
		tlsH2CPrivatePort := core.ServicePort{
			Name:        "h2c-tls-private",
			Port:        env.Spec.Providers.Web.TLS.H2CPrivatePort,
			Protocol:    "TCP",
			AppProtocol: &h2cAppProtocol,
			TargetPort:  intstr.FromInt(int(env.Spec.Providers.Web.TLS.H2CPrivatePort)),
		}
		servicePorts = append(servicePorts, tlsH2CPrivatePort)
		privH2CTLS = true
		privH2CPort = int32(env.Spec.Providers.Web.TLS.H2CPrivatePort)
	}

	if privTLS || pubTLS || pubH2CTLS || privH2CTLS {
		if err := generateCaddyConfigMap(cache, nn, app, pubTLS, privTLS, pubPort, privPort, pubH2CTLS, privH2CTLS, pubH2CPort, privH2CPort, env); err != nil {
			return err
		}
		populateSideCar(d, nn.Name, env.Spec.Providers.Web.TLS.Port, env.Spec.Providers.Web.TLS.PrivatePort, env.Spec.Providers.Web.TLS.H2CPort, env.Spec.Providers.Web.TLS.H2CPrivatePort, pubTLS, privTLS, pubH2CTLS, privH2CTLS, env)
		setServiceTLSAnnotations(s, nn.Name)
	}

	utils.MakeService(s, nn, map[string]string{"pod": nn.Name}, servicePorts, app, env.IsNodePort())

	d.Spec.Template.Spec.Containers[0].Ports = containerPorts

	if err := cache.Update(CoreService, s); err != nil {
		return err
	}

	return cache.Update(deployProvider.CoreDeployment, d)
}

func generateCaddyConfigMap(cache *rc.ObjectCache, nn types.NamespacedName, app *crd.ClowdApp, pub bool, priv bool, pubPort int32, privPort int32, pubH2C bool, privH2C bool, pubH2CPort int32, privH2CPort int32, env *crd.ClowdEnvironment) error {

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

	cmData, err := generateCaddyConfig(pub, priv, pubPort, privPort, pubH2C, privH2C, pubH2CPort, privH2CPort, env)
	if err != nil {
		return err
	}

	cm.Data = map[string]string{
		"caddy.json": cmData,
	}
	return cache.Update(CoreCaddyConfigMap, cm)
}

func populateSideCar(d *apps.Deployment, name string, port int32, privatePort int32, h2cPort int32, h2cPrivatePort int32, pub bool, priv bool, pubH2C bool, privH2C bool, env *crd.ClowdEnvironment) {
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
	if pubH2C {
		ports = append(ports, core.ContainerPort{
			Name:          "h2c-tls",
			ContainerPort: h2cPort,
			Protocol:      core.ProtocolTCP,
		})
	}
	if privH2C {
		ports = append(ports, core.ContainerPort{
			Name:          "h2c-tls-private",
			ContainerPort: h2cPrivatePort,
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
