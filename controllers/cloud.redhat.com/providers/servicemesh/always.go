package servicemesh

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	deployProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"
)

type servicemeshProvider struct {
	p.Provider
}

// NewServiceMeshProvider returns a new End provider run at the end of the provider set.
func NewServiceMeshProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &servicemeshProvider{Provider: *p}, nil
}

func (ch *servicemeshProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if ch.Env.Spec.Providers.ServiceMesh.Mode != "enabled" {
		return nil
	}

	dList := apps.DeploymentList{}
	if err := ch.Cache.List(deployProvider.CoreDeployment, &dList); err != nil {
		return err
	}

	for _, deployment := range dList.Items {
		annotations := deployment.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["sidecar.istio.io/inject"] = "true"

		deployment.Spec.Template.SetAnnotations(annotations)

		ch.Cache.Update(deployProvider.CoreDeployment, &deployment)
	}

	return nil
}
