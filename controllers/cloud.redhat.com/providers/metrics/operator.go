package metrics

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowder_config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type metricsProvider struct {
	providers.Provider
}

var PrometheusInstance = providers.NewSingleResourceIdent(ProvName, "prometheus_instance", &prom.Prometheus{})

var PrometheusRole = providers.NewSingleResourceIdent(ProvName, "prometheus_role", &rbac.Role{})

var PrometheusRoleBinding = providers.NewSingleResourceIdent(ProvName, "prometheus_role_binding", &rbac.RoleBinding{})

var PrometheusServiceAccount = providers.NewSingleResourceIdent(ProvName, "prometheus_service_account", &core.ServiceAccount{})

func NewMetricsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	if !p.Env.Spec.Providers.Metrics.Prometheus.Deploy {
		return &metricsProvider{Provider: *p}, nil
	}

	promObj := &prom.Prometheus{}
	nn := types.NamespacedName{
		Name:      p.Env.Name,
		Namespace: p.Env.Status.TargetNamespace,
	}

	if err := createPrometheusServiceAccount(p.Cache, p.Env); err != nil {
		return nil, err
	}

	if err := p.Cache.Create(PrometheusInstance, nn, promObj); err != nil {
		return nil, err
	}

	promObj.SetName(nn.Name)
	promObj.SetNamespace(nn.Namespace)
	promObj.Spec.ServiceMonitorSelector = &v1.LabelSelector{
		MatchLabels: map[string]string{
			"prometheus": p.Env.Name,
		},
	}
	promObj.Spec.ServiceAccountName = "prometheus"

	labeler := utils.GetCustomLabeler(map[string]string{"env": p.Env.Name}, nn, p.Env)
	labeler(promObj)

	if err := p.Cache.Update(PrometheusInstance, promObj); err != nil {
		return nil, err
	}

	return &metricsProvider{Provider: *p}, nil
}

func (m *metricsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if !app.PreHookDone() {
		return nil
	}

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, c); err != nil {
		return err
	}

	if clowder_config.LoadedConfig.Features.CreateServiceMonitor {
		if err := createServiceMonitorObjects(m.Cache, m.Env, app, c, m.Env.Name, m.Env.Status.TargetNamespace); err != nil {
			return err
		}

		if err := createPrometheusRoleBinding(m.Cache, app, m.Env); err != nil {
			return err
		}
	}

	return nil
}

func createPrometheusServiceAccount(cache *providers.ObjectCache, env *crd.ClowdEnvironment) error {

	cr := &core.ServiceAccount{}

	nn := types.NamespacedName{
		Name:      "prometheus",
		Namespace: env.Status.TargetNamespace,
	}

	if err := cache.Create(PrometheusServiceAccount, nn, cr); err != nil {
		return err
	}

	labeler := utils.GetCustomLabeler(map[string]string{}, nn, env)
	labeler(cr)

	if err := cache.Update(PrometheusServiceAccount, cr); err != nil {
		return err
	}

	return nil
}

func createPrometheusRoleBinding(cache *providers.ObjectCache, app *crd.ClowdApp, env *crd.ClowdEnvironment) error {

	crb := &rbac.RoleBinding{}

	nn := types.NamespacedName{
		Name:      app.Name,
		Namespace: app.Namespace,
	}

	if err := cache.Create(PrometheusRoleBinding, nn, crb); err != nil {
		return err
	}

	crb.RoleRef = rbac.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "clowder-prometheus",
	}

	crb.Subjects = []rbac.Subject{{
		Kind:      rbac.ServiceAccountKind,
		Name:      "prometheus",
		Namespace: env.GetClowdNamespace(),
	}}

	labeler := utils.GetCustomLabeler(map[string]string{}, nn, app)
	labeler(crb)

	if err := cache.Update(PrometheusRoleBinding, crb); err != nil {
		return err
	}

	return nil
}
