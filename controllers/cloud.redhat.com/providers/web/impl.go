package web

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

func (web *webProvider) makeService(deployment *crd.Deployment, app *crd.ClowdApp) error {

	s := &core.Service{}
	nn := app.GetDeploymentNamespacedName(deployment)

	if err := web.Cache.Create(CoreService, nn, s); err != nil {
		return err
	}

	d := &apps.Deployment{}

	web.Cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(deployment))

	servicePorts := []core.ServicePort{}
	containerPorts := []core.ContainerPort{}

	appProtocol := "http"
	if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
		// Create the core service port
		webPort := core.ServicePort{
			Name:        "public",
			Port:        web.Env.Spec.Providers.Web.Port,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
		}

		servicePorts = append(servicePorts, webPort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "web",
				ContainerPort: web.Env.Spec.Providers.Web.Port,
			},
		)

		if web.Env.Spec.Providers.AuthSidecar {
			authPort := core.ServicePort{
				Name:        "auth",
				Port:        8080,
				Protocol:    "TCP",
				AppProtocol: &appProtocol,
			}
			servicePorts = append(servicePorts, authPort)
		}
	}

	if deployment.WebServices.Private.Enabled {
		privatePort := web.Env.Spec.Providers.Web.PrivatePort

		if privatePort == 0 {
			privatePort = 10000
		}

		webPort := core.ServicePort{
			Name:        "private",
			Port:        privatePort,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
		}
		servicePorts = append(servicePorts, webPort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "private",
				ContainerPort: privatePort,
			},
		)
	}

	utils.MakeService(s, nn, map[string]string{"pod": nn.Name}, servicePorts, app, web.Env.IsNodePort())

	d.Spec.Template.Spec.Containers[0].Ports = containerPorts

	if err := web.Cache.Update(CoreService, s); err != nil {
		return err
	}

	if err := web.Cache.Update(deployProvider.CoreDeployment, d); err != nil {
		return err
	}

	return nil
}
