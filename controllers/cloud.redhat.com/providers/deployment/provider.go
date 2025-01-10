package deployment

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProvName sets the provider name identifier
var ProvName = "deployment"

// GetEnd returns the correct end provider.
func GetDeployment(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewDeploymentProvider(c)
}

func init() {
	providers.ProvidersRegistration.Register(GetDeployment, 0, ProvName)
}

func GetPodTemplateFromObject(deployment *crd.Deployment, rc *rc.ObjectCache, nn types.NamespacedName) (*core.PodTemplateSpec, error) {
	obj, err := GetClientObject(deployment, rc, nn)
	if err != nil {
		return nil, err
	}

	switch v := obj.(type) {
	case *apps.StatefulSet:
		return &v.Spec.Template, nil
	case *apps.Deployment:
		return &v.Spec.Template, nil
	}
	return nil, fmt.Errorf("no valid type")
}

func GetClientObject(deployment *crd.Deployment, rc *rc.ObjectCache, nn types.NamespacedName) (client.Object, error) {
	if deployment.Stateful.Enabled {
		ss := &apps.StatefulSet{}
		if err := rc.Get(CoreStatefulSet, ss, nn); err != nil {
			return nil, err
		}
		return ss, nil
	}

	d := &apps.Deployment{}
	if err := rc.Get(CoreDeployment, d, nn); err != nil {
		return nil, err
	}
	return d, nil
}

func UpdatePodTemplate(deployment *crd.Deployment, podTemplate *core.PodTemplateSpec, rc *rc.ObjectCache, nn types.NamespacedName) error {

	if deployment.Stateful.Enabled {
		ss := &apps.StatefulSet{}
		if err := rc.Get(CoreStatefulSet, ss, nn); err != nil {
			return err
		}
		ss.Spec.Template = *podTemplate
		if err := rc.Update(CoreStatefulSet, ss); err != nil {
			return err
		}
	}

	d := &apps.Deployment{}
	if err := rc.Get(CoreDeployment, d, nn); err != nil {
		return err
	}
	d.Spec.Template = *podTemplate
	if err := rc.Update(CoreDeployment, d); err != nil {
		return err
	}

	return nil
}
