package autoscaler

import (
	"log"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type simpleAutoScalerProvider struct {
	providers.Provider
	Config config.DatabaseConfig
}

// NewNoneDBProvider returns a new none db provider object.
func NewSimpleAutoScalerProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &simpleAutoScalerProvider{Provider: *p}, nil
}

func (sp *simpleAutoScalerProvider) Provide(app *crd.ClowdApp, appConfig *config.AppConfig) error {
	for _, deployment := range app.Spec.Deployments {

		// Create the autoscaler if one is defined
		if deployment.SimpleAutoScaler != nil {
			if err := sp.makeAutoScalers(&deployment, app, appConfig); err != nil {
				return err
			}
		}

	}
	return nil
}

func (sp *simpleAutoScalerProvider) makeAutoScalers(deployment *crd.Deployment, app *crd.ClowdApp, appConfig *config.AppConfig) error {

	coreDeployment, err := sp.getCoreDeployment(deployment, app)
	if err != nil {
		return err
	}
	hpa := sp.createHorizontalPodAutoscaler(coreDeployment, app, appConfig)
	erree := sp.Provider.Client.Create(sp.Ctx, hpa)
	if erree != nil {
		log.Println(erree)
	}
	return nil
}

func (sp *simpleAutoScalerProvider) createHorizontalPodAutoscaler(coreDeployment *apps.Deployment, app *crd.ClowdApp, appConfig *config.AppConfig) *v2.HorizontalPodAutoscaler {
	var minReplicas int32 = 1
	var maxReplicas int32 = 10

	ref := metav1.OwnerReference{
		APIVersion: coreDeployment.TypeMeta.APIVersion,
		Kind:       coreDeployment.TypeMeta.Kind,
		Name:       coreDeployment.ObjectMeta.Name,
		UID:        coreDeployment.ObjectMeta.UID,
	}

	hpa := &v2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2beta2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name + "-" + coreDeployment.Name + "-" + "hpa",
			Namespace: coreDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				ref,
			},
		},
		Spec: v2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2.CrossVersionObjectReference{
				APIVersion: coreDeployment.APIVersion,
				Kind:       coreDeployment.Kind,
				Name:       coreDeployment.Name,
			},
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []v2.MetricSpec{
				sp.makeMetricSpec(v1.ResourceCPU, v2.MetricTargetType("Utilization"), 60),
				sp.makeMetricSpec(v1.ResourceMemory, v2.MetricTargetType("Utilization"), 60),
			},
		},
	}
	return hpa
}

func (sp *simpleAutoScalerProvider) makeMetricSpec(resource v1.ResourceName, targetType v2.MetricTargetType, target int32) v2.MetricSpec {
	ms := &v2.MetricSpec{
		Type: v2.MetricSourceType("Resource"),
		Resource: &v2.ResourceMetricSource{
			Name: resource,
			Target: v2.MetricTarget{
				Type:               targetType,
				AverageUtilization: &target,
			},
		},
	}
	return *ms
}

func (sp *simpleAutoScalerProvider) getCoreDeployment(clowdDeployment *crd.Deployment, app *crd.ClowdApp) (*apps.Deployment, error) {
	nn := app.GetDeploymentNamespacedName(clowdDeployment)
	d := &apps.Deployment{}
	if err := sp.Cache.Get(deployProvider.CoreDeployment, d, nn); err != nil {
		return d, err
	}
	return d, nil
}
