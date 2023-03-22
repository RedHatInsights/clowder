package metrics

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	webProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/web"

	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

func makeMetrics(cache *rc.ObjectCache, deployment *crd.Deployment, app *crd.ClowdApp, port int32) error {

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
		Name:        "metrics",
		Port:        port,
		Protocol:    "TCP",
		AppProtocol: &appProtocol,
		TargetPort:  intstr.FromInt(int(port)),
	}

	s.Spec.Ports = append(s.Spec.Ports, metricsPort)

	d.Spec.Template.Spec.Containers[0].Ports = append(d.Spec.Template.Spec.Containers[0].Ports,
		core.ContainerPort{
			Name:          "metrics",
			ContainerPort: port,
			Protocol:      core.ProtocolTCP,
		},
	)

	if err := cache.Update(webProvider.CoreService, s); err != nil {
		return err
	}

	return cache.Update(deployProvider.CoreDeployment, d)
}

func createMetricsOnDeployments(cache *rc.ObjectCache, env *crd.ClowdEnvironment, app *crd.ClowdApp, c *config.AppConfig) error {
	c.MetricsPort = int(env.Spec.Providers.Metrics.Port)
	c.MetricsPath = env.Spec.Providers.Metrics.Path

	for _, deployment := range app.Spec.Deployments {
		innerDeployment := deployment
		if err := makeMetrics(cache, &innerDeployment, app, env.Spec.Providers.Metrics.Port); err != nil {
			return err
		}
	}

	return nil
}

func createServiceMonitorObjects(cache *rc.ObjectCache, env *crd.ClowdEnvironment, app *crd.ClowdApp, promLabel string, namespace string) error {
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

		labeler := utils.GetCustomLabeler(map[string]string{"prometheus": promLabel}, nn, env)
		labeler(sm)

		sm.SetNamespace(namespace)

		if err := cache.Update(MetricsServiceMonitor, sm); err != nil {
			return err
		}
	}
	return nil
}
