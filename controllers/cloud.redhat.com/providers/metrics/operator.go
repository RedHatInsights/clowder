package metrics

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	sub "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/metrics/subscriptions"

	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type metricsProvider struct {
	providers.Provider
}

var PrometheusSubscription = rc.NewSingleResourceIdent(ProvName, "prometheus_subscription", &sub.Subscription{})

var PrometheusInstance = rc.NewSingleResourceIdent(ProvName, "prometheus_instance", &prom.Prometheus{})

var PrometheusRole = rc.NewSingleResourceIdent(ProvName, "prometheus_role", &rbac.Role{})

var PrometheusRoleBinding = rc.NewSingleResourceIdent(ProvName, "prometheus_role_binding", &rbac.RoleBinding{})

var PrometheusServiceAccount = rc.NewSingleResourceIdent(ProvName, "prometheus_service_account", &core.ServiceAccount{})

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
	promObj.Spec.ServiceMonitorSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"prometheus": p.Env.Name,
		},
	}
	promObj.Spec.ServiceAccountName = "prometheus"
	promObj.Spec.Resources = core.ResourceRequirements{
		Limits: core.ResourceList{
			"memory": resource.MustParse("400Mi"),
			"cpu":    resource.MustParse("400m"),
		},
		Requests: core.ResourceList{
			"memory": resource.MustParse("200Mi"),
			"cpu":    resource.MustParse("50m"),
		},
	}

	labeler := utils.GetCustomLabeler(map[string]string{"env": p.Env.Name}, nn, p.Env)
	labeler(promObj)

	if err := p.Cache.Update(PrometheusInstance, promObj); err != nil {
		return nil, err
	}

	if err := createSubscription(p.Cache, p.Env); err != nil {
		return nil, err
	}

	return &metricsProvider{Provider: *p}, nil
}

func (m *metricsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, c); err != nil {
		return err
	}

	if clowderconfig.LoadedConfig.Features.CreateServiceMonitor {
		if err := createServiceMonitorObjects(m.Cache, m.Env, app, c, m.Env.Name, m.Env.Status.TargetNamespace); err != nil {
			return err
		}

		if err := createPrometheusRoleBinding(m.Cache, app, m.Env); err != nil {
			return err
		}
	}

	return nil
}

func createPrometheusServiceAccount(cache *rc.ObjectCache, env *crd.ClowdEnvironment) error {

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

func createPrometheusRoleBinding(cache *rc.ObjectCache, app *crd.ClowdApp, env *crd.ClowdEnvironment) error {

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

func createSubscription(cache *rc.ObjectCache, env *crd.ClowdEnvironment) error {

	subObj := &sub.Subscription{}

	nn := types.NamespacedName{
		Name:      "prometheus",
		Namespace: env.Status.TargetNamespace,
	}

	if err := cache.Create(PrometheusSubscription, nn, subObj); err != nil {
		return err
	}

	labeler := utils.GetCustomLabeler(map[string]string{"env": env.Name}, nn, env)
	labeler(subObj)
	subObj.SetOwnerReferences([]metav1.OwnerReference{env.MakeOwnerReference()})

	subObj.Spec = &sub.SubscriptionSpec{
		Channel:             utils.StringPtr("beta"),
		InstallPlanApproval: utils.StringPtr("Automatic"),
		Name:                "prometheus",
		Source:              "community-operators",
		SourceNamespace:     "openshift-marketplace",
		StartingCSV:         utils.StringPtr("prometheusoperator.0.47.0"),
	}

	if err := cache.Update(PrometheusSubscription, subObj); err != nil {
		return err
	}

	return nil
}
