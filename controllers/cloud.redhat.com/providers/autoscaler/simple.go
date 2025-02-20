package autoscaler

import (
	"fmt"

	res "k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	v2 "k8s.io/api/autoscaling/v2"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClowdAPIVersion      = "clowd.redhat.com/v1alpha1"
	ClowdKind            = "ClowdApp"
	DeploymentAPIVersion = "apps/v1"
	DeploymentKind       = "Deployment"
)

// Creates a simple HPA in the resource cache for the deployment and ClowdApp
func ProvideSimpleAutoScaler(app *crd.ClowdApp, appConfig *config.AppConfig, sp *providers.Provider, deployment crd.Deployment) error {
	coreObject, err := getcoreObjectFromCache(&deployment, app, sp)
	if err != nil {
		return errors.Wrap("Could not get deployment from resource cache", err)
	}
	hpaMaker := newSimpleHPAMaker(&deployment, app, appConfig, coreObject)
	hpaResource := hpaMaker.getResource()

	err = cacheAutoscaler(app, sp, deployment, hpaResource)
	if err != nil {
		return errors.Wrap("Could not add HPA to resource cache", err)
	}

	return nil
}

// Adds the HPA to the resource cache
func cacheAutoscaler(app *crd.ClowdApp, sp *providers.Provider, deployment crd.Deployment, hpaResource v2.HorizontalPodAutoscaler) error {
	nn := app.GetDeploymentNamespacedName(&deployment)
	return sp.Cache.Create(SimpleAutoScaler, nn, &hpaResource)
}

// Get the core apps.Deployment from the provider cache
func getcoreObjectFromCache(clowdDeployment *crd.Deployment, app *crd.ClowdApp, sp *providers.Provider) (client.Object, error) {
	nn := app.GetDeploymentNamespacedName(clowdDeployment)

	obj, err := deployProvider.GetClientObject(clowdDeployment, sp.Cache, nn)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Factory for the simpleHPAMaker
func newSimpleHPAMaker(deployment *crd.Deployment, app *crd.ClowdApp, appConfig *config.AppConfig, coreObject client.Object) simpleHPAMaker {
	return simpleHPAMaker{
		deployment: deployment,
		app:        app,
		appConfig:  appConfig,
		coreObject: coreObject,
	}
}

// Creates a simple HPA and stores references
// to the resources and dependencies it requires
type simpleHPAMaker struct {
	deployment *crd.Deployment
	app        *crd.ClowdApp
	appConfig  *config.AppConfig
	coreObject client.Object
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
			Namespace: d.coreObject.GetNamespace(),
		},
		Spec: v2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2.CrossVersionObjectReference{
				APIVersion: DeploymentAPIVersion,
				Kind:       DeploymentKind,
				Name:       d.coreObject.GetName(),
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
		metricsSpec := d.makeAverageUtilizationMetricSpec(core.ResourceMemory, d.deployment.AutoScalerSimple.RAM.ScaleAtUtilization)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}
	if d.deployment.AutoScalerSimple.RAM.ScaleAtValue != "" {
		threshold := res.MustParse(d.deployment.AutoScalerSimple.RAM.ScaleAtValue)
		metricsSpec := d.makeAverageValueMetricSpec(core.ResourceMemory, threshold)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}

	if d.deployment.AutoScalerSimple.CPU.ScaleAtUtilization != 0 {
		metricsSpec := d.makeAverageUtilizationMetricSpec(core.ResourceCPU, d.deployment.AutoScalerSimple.CPU.ScaleAtUtilization)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}
	if d.deployment.AutoScalerSimple.CPU.ScaleAtValue != "" {
		threshold := res.MustParse(d.deployment.AutoScalerSimple.CPU.ScaleAtValue)
		metricsSpec := d.makeAverageValueMetricSpec(core.ResourceCPU, threshold)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}

	return metricsSpecs
}

func (d *simpleHPAMaker) makeAverageValueMetricSpec(resource core.ResourceName, threshold res.Quantity) v2.MetricSpec {
	ms := d.makeBasicMetricSpec(resource)
	ms.Resource.Target.Type = v2.AverageValueMetricType
	ms.Resource.Target.AverageValue = &threshold
	return ms
}

func (d *simpleHPAMaker) makeAverageUtilizationMetricSpec(resource core.ResourceName, threshold int32) v2.MetricSpec {
	ms := d.makeBasicMetricSpec(resource)
	ms.Resource.Target.Type = v2.UtilizationMetricType
	ms.Resource.Target.AverageUtilization = &threshold
	return ms
}

func (d *simpleHPAMaker) makeBasicMetricSpec(resource core.ResourceName) v2.MetricSpec {
	ms := v2.MetricSpec{
		Type: v2.MetricSourceType("Resource"),
		Resource: &v2.ResourceMetricSource{
			Name:   resource,
			Target: v2.MetricTarget{},
		},
	}
	return ms
}
