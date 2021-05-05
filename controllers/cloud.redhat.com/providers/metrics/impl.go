package metrics

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	deployProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"
	webProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/web"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func makeMetrics(cache *providers.ObjectCache, deployment *crd.Deployment, app *crd.ClowdApp, port int32) error {

	s := &core.Service{}

	if err := cache.Get(webProvider.CoreService, s, app.GetDeploymentNamespacedName(deployment)); err != nil {
		return err
	}

	d := &apps.Deployment{}

	if err := cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(deployment)); err != nil {
		return err
	}

	appProtocol := "http"
	metricsPort := core.ServicePort{
		Name: "metrics", Port: port, Protocol: "TCP", AppProtocol: &appProtocol,
	}

	s.Spec.Ports = append(s.Spec.Ports, metricsPort)

	d.Spec.Template.Spec.Containers[0].Ports = append(d.Spec.Template.Spec.Containers[0].Ports,
		core.ContainerPort{
			Name:          "metrics",
			ContainerPort: port,
		},
	)

	if err := cache.Update(webProvider.CoreService, s); err != nil {
		return err
	}

	if err := cache.Update(deployProvider.CoreDeployment, d); err != nil {
		return err
	}

	return nil
}

func createMetricsOnDeployments(cache *providers.ObjectCache, env *crd.ClowdEnvironment, app *crd.ClowdApp, c *config.AppConfig) error {
	c.MetricsPort = int(env.Spec.Providers.Metrics.Port)
	c.MetricsPath = env.Spec.Providers.Metrics.Path

	for _, deployment := range app.Spec.Deployments {

		if err := makeMetrics(cache, &deployment, app, env.Spec.Providers.Metrics.Port); err != nil {
			return err
		}
	}

	return nil
}

func createServiceMonitorObjects(cache *providers.ObjectCache, env *crd.ClowdEnvironment, app *crd.ClowdApp, c *config.AppConfig, promLabel string, namespace string) error {
	for _, deployment := range app.Spec.Deployments {
		sm := &prom.ServiceMonitor{}
		name := fmt.Sprintf("%s-%s", app.Name, deployment.Name)

		nn := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}

		if err := cache.Create(MetricsServiceMonitor, nn, sm); err != nil {
			return err
		}
		sm.Spec.Endpoints = []prom.Endpoint{{
			Interval: "15s",
			Path:     env.Spec.Providers.Metrics.Path,
			Port:     "metrics",
		}}

		sm.Spec.NamespaceSelector = prom.NamespaceSelector{
			MatchNames: []string{app.Namespace},
		}

		sm.Spec.Selector = v1.LabelSelector{
			MatchLabels: map[string]string{
				"pod": nn.Name,
			},
		}

		var labeler func(v1.Object)

		labeler = utils.GetCustomLabeler(map[string]string{"prometheus": promLabel}, nn, env)
		labeler(sm)

		sm.SetNamespace(namespace)

		if err := cache.Update(MetricsServiceMonitor, sm); err != nil {
			return err
		}
	}
	return nil
}
