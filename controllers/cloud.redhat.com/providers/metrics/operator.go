package metrics

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	sub "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/metrics/subscriptions"

	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
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
	p.Cache.AddPossibleGVKFromIdent(
		PrometheusSubscription,
		PrometheusInstance,
		PrometheusRole,
		PrometheusRoleBinding,
		PrometheusServiceAccount,
	)
	return &metricsProvider{Provider: *p}, nil
}

func (m *metricsProvider) EnvProvide() error {
	if !m.Env.Spec.Providers.Metrics.Prometheus.Deploy {
		return nil
	}

	promObj := &prom.Prometheus{}
	nn := types.NamespacedName{
		Name:      m.Env.Name,
		Namespace: m.Env.Status.TargetNamespace,
	}

	if err := createPrometheusServiceAccount(m.Cache, m.Env); err != nil {
		return err
	}

	if err := m.Cache.Create(PrometheusInstance, nn, promObj); err != nil {
		return err
	}

	promObj.SetName(nn.Name)
	promObj.SetNamespace(nn.Namespace)
	promObj.Spec.ServiceMonitorSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"prometheus": m.Env.Name,
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

	labeler := utils.GetCustomLabeler(map[string]string{"env": m.Env.Name}, nn, m.Env)
	labeler(promObj)

	if err := m.Cache.Update(PrometheusInstance, promObj); err != nil {
		return err
	}

	return createSubscription(m.Cache, m.Env)
}

func (m *metricsProvider) Provide(app *crd.ClowdApp) error {

	if err := createMetricsOnDeployments(m.Cache, m.Env, app, m.Config); err != nil {
		return err
	}

	if clowderconfig.LoadedConfig.Features.CreateServiceMonitor {
		if err := createServiceMonitorObjects(m.Cache, m.Env, app, m.Env.Name, m.Env.Status.TargetNamespace); err != nil {
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

	return cache.Update(PrometheusServiceAccount, cr)
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

	return cache.Update(PrometheusRoleBinding, crb)
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

	return cache.Update(PrometheusSubscription, subObj)
}
