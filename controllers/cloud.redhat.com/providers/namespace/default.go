package namespace

import (
	"io/ioutil"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type namespaceProvider struct {
	providers.Provider
}

// NewNamespaceProvider returns a new Namespace provider.
func NewNamespaceProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	clowderNsB, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")

	if err != nil {
		return nil, err
	}

	clowderNs := string(clowderNsB)

	err = setLabelOnNamespace(p, clowderNs)

	if err != nil {
		return nil, err
	}

	return &namespaceProvider{Provider: *p}, setLabelOnNamespace(p, p.Env.Status.TargetNamespace)
}

func (nsp *namespaceProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	return setLabelOnNamespace(&nsp.Provider, app.GetNamespace())
}

func setLabelOnNamespace(p *providers.Provider, ns string) error {
	nsType := &core.Namespace{}
	err := p.Client.Get(p.Ctx, types.NamespacedName{Name: ns}, nsType)

	if err != nil {
		return err
	}

	annotations := nsType.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
	}

	if _, ok := annotations["kubernetes.io/metadata.name"]; !ok {
		annotations["kubernetes.io/metadata.name"] = ns
		return p.Client.Update(p.Ctx, nsType)
	}

	return nil
}
