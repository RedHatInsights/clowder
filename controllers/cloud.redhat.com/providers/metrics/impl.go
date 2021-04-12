package metrics

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	deployProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"
	webProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/web"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

func (m *metricsProvider) makeMetrics(deployment *crd.Deployment, app *crd.ClowdApp) error {

	s := &core.Service{}

	if err := m.Cache.Get(webProvider.CoreService, s, app.GetDeploymentNamespacedName(deployment)); err != nil {
		return err
	}

	d := &apps.Deployment{}

	if err := m.Cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(deployment)); err != nil {
		return err
	}

	appProtocol := "http"
	metricsPort := core.ServicePort{
		Name: "metrics", Port: m.Env.Spec.Providers.Metrics.Port, Protocol: "TCP", AppProtocol: &appProtocol,
	}

	s.Spec.Ports = append(s.Spec.Ports, metricsPort)

	d.Spec.Template.Spec.Containers[0].Ports = append(d.Spec.Template.Spec.Containers[0].Ports,
		core.ContainerPort{
			Name:          "metrics",
			ContainerPort: m.Env.Spec.Providers.Metrics.Port,
		},
	)

	if err := m.Cache.Update(webProvider.CoreService, s); err != nil {
		return err
	}

	if err := m.Cache.Update(deployProvider.CoreDeployment, d); err != nil {
		return err
	}

	return nil
}
