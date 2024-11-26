package statefulset

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	apps "k8s.io/api/apps/v1"
	"fmt"
)

type statefulSetProvider struct {
	providers.Provider
}

// CoreStatefulSet is the statefulset object for the apps deployments.
var CoreStatefulSet = rc.NewMultiResourceIdent(ProvName, "core_statefulset", &apps.StatefulSet{})

func NewStatefulSetProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(CoreStatefulSet)
	return &statefulSetProvider{Provider: *p}, nil
}

func (dp *statefulSetProvider) EnvProvide() error {
	return nil
}

func (dp *statefulSetProvider) Provide(app *crd.ClowdApp) error {

	for _, deployment := range app.Spec.Deployments {
		fmt.Printf("statefulset provider checking deployment %s", deployment.Name)	
		if deployment.UseStatefulSet {
			fmt.Printf("statefulset provider processing deployment %s", deployment.Name)
			if err := dp.makeStatefulSet(deployment, app); err != nil {
				return err
			}
		}
	}
	return nil
}
