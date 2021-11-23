package job

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	batchv1 "k8s.io/api/batch/v1"
)

type jobProvider struct {
	p.Provider
}

var PreHookJob = p.NewSingleResourceIdent(ProvName, "pre_hook_job", &batchv1.Job{})

func NewJobProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &jobProvider{Provider: *p}, nil
}

func (jp *jobProvider) Provide(app *crd.ClowdApp, _ *config.AppConfig) error {
	if app.RunPreHook() {
		if err := jp.makePreHookJob(&app.Spec.PreHookJob, app); err != nil {
			return err
		}
	} else if app.PreHookDone() {
		fmt.Println("YEET")
	}

	return nil
}
