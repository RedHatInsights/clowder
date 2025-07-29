// Package namespace provides namespace management functionality for Clowder applications
package namespace

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	utils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type namespaceProvider struct {
	providers.Provider
}

// NewNamespaceProvider returns a new Namespace provider.
func NewNamespaceProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &namespaceProvider{Provider: *p}, nil
}

func (nsp *namespaceProvider) EnvProvide() error {
	clowderNs, nSerr := utils.GetClowderNamespace()

	if nSerr == nil {
		// CLOBBER: Purposefully ignoring the error here
		_ = setLabelOnNamespace(&nsp.Provider, clowderNs)
	}

	return setLabelOnNamespace(&nsp.Provider, nsp.Env.Status.TargetNamespace)
}

func (nsp *namespaceProvider) Provide(app *crd.ClowdApp) error {
	return setLabelOnNamespace(&nsp.Provider, app.GetNamespace())
}

func setLabelOnNamespace(p *providers.Provider, ns string) error {
	nsType := &core.Namespace{}
	err := p.Client.Get(p.Ctx, types.NamespacedName{Name: ns}, nsType)

	if err != nil {
		return err
	}

	labels := nsType.GetLabels()

	if labels == nil {
		labels = make(map[string]string)
	}

	if _, ok := labels["kubernetes.io/metadata.name"]; !ok {
		labels["kubernetes.io/metadata.name"] = ns
		nsType.SetLabels(labels)
		return p.Client.Update(p.Ctx, nsType)
	}

	return nil
}
