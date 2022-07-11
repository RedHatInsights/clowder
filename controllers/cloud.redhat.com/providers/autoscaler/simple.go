package autoscaler

import (
	res "k8s.io/apimachinery/pkg/api/resource"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ProvideSimpleAutoScaler(app *crd.ClowdApp, appConfig *config.AppConfig, sp *providers.Provider, deployment crd.Deployment) error {
	cachedDeployment, err := getDeploymentFromCache(&deployment, app, sp)
	if err != nil {
		return errors.Wrap("Could not get deployment from cache", err)
	}

	deploymentHPA := makeDeploymentSimpleHPA(&deployment, app, appConfig, cachedDeployment)
	hpaResource := deploymentHPA.getResource()

	err = sp.Client.Create(sp.Ctx, &hpaResource)

	if err != nil {
		return errors.Wrap("Could not create HPA resource", err)
	}

	return err
}

//Get the core apps.Deployment from the provider cache
//This is in AutoScalerSimpleProvider because we need access to the provider cache
func getDeploymentFromCache(clowdDeployment *crd.Deployment, app *crd.ClowdApp, sp *providers.Provider) (*apps.Deployment, error) {
	nn := app.GetDeploymentNamespacedName(clowdDeployment)
	d := &apps.Deployment{}
	if err := sp.Cache.Get(deployProvider.CoreDeployment, d, nn); err != nil {
		return d, err
	}
	return d, nil
}

func makeDeploymentSimpleHPA(deployment *crd.Deployment, app *crd.ClowdApp, appConfig *config.AppConfig, coreDeployment *apps.Deployment) deployemntSimpleHPA {
	return deployemntSimpleHPA{
		deployment:     deployment,
		app:            app,
		appConfig:      appConfig,
		coreDeployment: coreDeployment,
	}
}

type deployemntSimpleHPA struct {
	deployment     *crd.Deployment
	app            *crd.ClowdApp
	appConfig      *config.AppConfig
	coreDeployment *apps.Deployment
}

func (d *deployemntSimpleHPA) getResource() v2.HorizontalPodAutoscaler {
	hpa := d.makeHPA()
	hpa.Spec.Metrics = d.makeMetricsSpecs()
	return hpa
}

func (d *deployemntSimpleHPA) makeHPA() v2.HorizontalPodAutoscaler {
	hpa := v2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.app.Name + "-" + d.coreDeployment.Name + "-" + "hpa",
			Namespace: d.coreDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: d.coreDeployment.TypeMeta.APIVersion,
					Kind:       d.coreDeployment.TypeMeta.Kind,
					Name:       d.coreDeployment.ObjectMeta.Name,
					UID:        d.coreDeployment.ObjectMeta.UID,
				},
			},
		},
		Spec: v2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2.CrossVersionObjectReference{
				APIVersion: d.coreDeployment.APIVersion,
				Kind:       d.coreDeployment.Kind,
				Name:       d.coreDeployment.Name,
			},
			MinReplicas: &d.deployment.AutoScalerSimple.Replicas.Min,
			MaxReplicas: d.deployment.AutoScalerSimple.Replicas.Max,
		},
	}
	return hpa
}

func (d *deployemntSimpleHPA) makeMetricsSpecs() []v2.MetricSpec {
	metricsSpecs := []v2.MetricSpec{}

	if d.deployment.AutoScalerSimple.RAM.ScaleAtUtilization != 0 {
		metricsSpec := d.makeAverageUtilizationMetricSpec(v1.ResourceMemory, d.deployment.AutoScalerSimple.RAM.ScaleAtUtilization)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}
	if d.deployment.AutoScalerSimple.RAM.ScaleAtValue != "" {
		threshhold := res.MustParse(d.deployment.AutoScalerSimple.RAM.ScaleAtValue)
		metricsSpec := d.makeAverageValueMetricSpec(v1.ResourceMemory, threshhold)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}

	if d.deployment.AutoScalerSimple.CPU.ScaleAtUtilization != 0 {
		metricsSpec := d.makeAverageUtilizationMetricSpec(v1.ResourceCPU, d.deployment.AutoScalerSimple.CPU.ScaleAtUtilization)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}
	if d.deployment.AutoScalerSimple.CPU.ScaleAtValue != "" {
		threshhold := res.MustParse(d.deployment.AutoScalerSimple.CPU.ScaleAtValue)
		metricsSpec := d.makeAverageValueMetricSpec(v1.ResourceCPU, threshhold)
		metricsSpecs = append(metricsSpecs, metricsSpec)
	}

	return metricsSpecs
}

func (d *deployemntSimpleHPA) makeAverageValueMetricSpec(resource v1.ResourceName, threshhold res.Quantity) v2.MetricSpec {
	ms := d.makeBasicMetricSpec(resource)
	ms.Resource.Target.Type = v2.AverageValueMetricType
	ms.Resource.Target.AverageValue = &threshhold
	return ms
}

func (d *deployemntSimpleHPA) makeAverageUtilizationMetricSpec(resource v1.ResourceName, threshhold int32) v2.MetricSpec {
	ms := d.makeBasicMetricSpec(resource)
	ms.Resource.Target.Type = v2.UtilizationMetricType
	ms.Resource.Target.AverageUtilization = &threshhold
	return ms
}

func (d *deployemntSimpleHPA) makeBasicMetricSpec(resource v1.ResourceName) v2.MetricSpec {
	ms := v2.MetricSpec{
		Type: v2.MetricSourceType("Resource"),
		Resource: &v2.ResourceMetricSource{
			Name:   resource,
			Target: v2.MetricTarget{},
		},
	}
	return ms
}
