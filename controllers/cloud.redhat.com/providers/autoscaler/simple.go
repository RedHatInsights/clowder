package autoscaler

import (
	"fmt"

	res "k8s.io/apimachinery/pkg/api/resource"

	apps "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
)

const (
	ClowdAPIVersion      = "clowd.redhat.com/v1alpha1"
	ClowdKind            = "ClowdApp"
	DeploymentAPIVersion = "apps/v1"
	DeploymentKind       = "Deployment"
)

// ProvideSimpleAutoScaler creates a simple HPA in the resource cache for the deployment and ClowdApp
func ProvideSimpleAutoScaler(app *crd.ClowdApp, appConfig *config.AppConfig, sp *providers.Provider, deployment *crd.Deployment) error {
	cachedDeployment, err := getDeploymentFromCache(deployment, app, sp)
	if err != nil {
		return errors.Wrap("Could not get deployment from resource cache", err)
	}
	hpaMaker := newSimpleHPAMaker(deployment, app, appConfig, cachedDeployment)
	hpaResource := hpaMaker.getResource()

	err = cacheAutoscaler(app, sp, deployment, &hpaResource)
	if err != nil {
		return errors.Wrap("Could not add HPA to resource cache", err)
	}

	return nil
}

// Adds the HPA to the resource cache
func cacheAutoscaler(app *crd.ClowdApp, sp *providers.Provider, deployment *crd.Deployment, hpaResource *v2.HorizontalPodAutoscaler) error {
	nn := app.GetDeploymentNamespacedName(deployment)
	return sp.Cache.Create(SimpleAutoScaler, nn, hpaResource)
}

// Get the core apps.Deployment from the provider cache
func getDeploymentFromCache(clowdDeployment *crd.Deployment, app *crd.ClowdApp, sp *providers.Provider) (*apps.Deployment, error) {
	nn := app.GetDeploymentNamespacedName(clowdDeployment)
	d := &apps.Deployment{}
	if err := sp.Cache.Get(deployProvider.CoreDeployment, d, nn); err != nil {
		return d, err
	}
	return d, nil
}

// Factory for the simpleHPAMaker
func newSimpleHPAMaker(deployment *crd.Deployment, app *crd.ClowdApp, appConfig *config.AppConfig, coreDeployment *apps.Deployment) simpleHPAMaker {
	return simpleHPAMaker{
		deployment:     deployment,
		app:            app,
		appConfig:      appConfig,
		coreDeployment: coreDeployment,
	}
}

// Creates a simple HPA and stores references
// to the resources and dependencies it requires
type simpleHPAMaker struct {
	deployment     *crd.Deployment
	app            *crd.ClowdApp
	appConfig      *config.AppConfig
	coreDeployment *apps.Deployment
}

// Constructs the HPA in 2 parts: the HPA itself and the metric spec
func (d *simpleHPAMaker) getResource() v2.HorizontalPodAutoscaler {
	hpa := d.makeHPA()
	hpa.Spec.Metrics = d.makeMetricsSpecs()
	return hpa
}

// Creates the HPA resource
func (d *simpleHPAMaker) makeHPA() v2.HorizontalPodAutoscaler {
	name := fmt.Sprintf("%s-%s-hpa", d.app.Name, d.deployment.Name)
	hpa := v2.HorizontalPodAutoscaler{
		// Set to clowdapp
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: ClowdAPIVersion,
					Kind:       ClowdKind,
					Name:       d.app.Name,
					UID:        d.app.UID,
				}},
			Name:      name,
			Namespace: d.coreDeployment.Namespace,
		},
		Spec: v2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2.CrossVersionObjectReference{
				APIVersion: DeploymentAPIVersion,
				Kind:       DeploymentKind,
				Name:       d.coreDeployment.Name,
			},
			MinReplicas: &d.deployment.AutoScalerSimple.Replicas.Min,
			MaxReplicas: d.deployment.AutoScalerSimple.Replicas.Max,
		},
	}
	return hpa
}

// Creates the metrics specs for the HPA
func (d *simpleHPAMaker) makeMetricsSpecs() []v2.MetricSpec {
	metricsSpecs := []v2.MetricSpec{}

	if d.deployment.AutoScalerSimple.RAM.ScaleAtUtilization != 0 {
		metricsSpec := d.makeAverageUtilizationMetricSpec(v1.ResourceMemory, d.deployment.AutoScalerSimple.RAM.ScaleAtUtilization)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}
	if d.deployment.AutoScalerSimple.RAM.ScaleAtValue != "" {
		threshold := res.MustParse(d.deployment.AutoScalerSimple.RAM.ScaleAtValue)
		metricsSpec := d.makeAverageValueMetricSpec(v1.ResourceMemory, threshold)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}

	if d.deployment.AutoScalerSimple.CPU.ScaleAtUtilization != 0 {
		metricsSpec := d.makeAverageUtilizationMetricSpec(v1.ResourceCPU, d.deployment.AutoScalerSimple.CPU.ScaleAtUtilization)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}
	if d.deployment.AutoScalerSimple.CPU.ScaleAtValue != "" {
		threshold := res.MustParse(d.deployment.AutoScalerSimple.CPU.ScaleAtValue)
		metricsSpec := d.makeAverageValueMetricSpec(v1.ResourceCPU, threshold)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}

	return metricsSpecs
}

func (d *simpleHPAMaker) makeAverageValueMetricSpec(resource v1.ResourceName, threshold res.Quantity) v2.MetricSpec {
	ms := d.makeBasicMetricSpec(resource)
	ms.Resource.Target.Type = v2.AverageValueMetricType
	ms.Resource.Target.AverageValue = &threshold
	return ms
}

func (d *simpleHPAMaker) makeAverageUtilizationMetricSpec(resource v1.ResourceName, threshold int32) v2.MetricSpec {
	ms := d.makeBasicMetricSpec(resource)
	ms.Resource.Target.Type = v2.UtilizationMetricType
	ms.Resource.Target.AverageUtilization = &threshold
	return ms
}

func (d *simpleHPAMaker) makeBasicMetricSpec(resource v1.ResourceName) v2.MetricSpec {
	ms := v2.MetricSpec{
		Type: v2.MetricSourceType("Resource"),
		Resource: &v2.ResourceMetricSource{
			Name:   resource,
			Target: v2.MetricTarget{},
		},
	}
	return ms
}
