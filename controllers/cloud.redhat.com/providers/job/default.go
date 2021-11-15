package job

import (
	"fmt"
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	batchv1 "k8s.io/api/batch/v1"
	"strconv"
)

type jobProvider struct {
	p.Provider
}

var PreHookJob = p.NewSingleResourceIdent(ProvName, "pre_hook_job", &batchv1.Job{})

func NewJobProvider(p *p.Provider) (p.ClowderProvider, error) {
	return &jobProvider{Provider: *p}, nil
}

func (jp *jobProvider) Provide(app *crd.ClowdApp, _ *config.AppConfig) error {
	if shouldRunPreHookJob(app) {
		if err := jp.makePreHookJob(&app.Spec.PreHookJob, app); err != nil {
			return err
		}
	} else if preHookDone(app) {
		fmt.Println("YEET")
	}

	return nil
}

func shouldRunPreHookJob(app *crd.ClowdApp) bool {
	// is there a better way to find out if the app has a job?
	if app.Spec.PreHookJob.Name == "" || app.Annotations["clowder/pre-hook-status"] == "pending" {
		return false
	} else if app.Annotations["clowder/pre-hook-generation"] == "" {
		return true
	}

	preHookGen, err := strconv.Atoi(app.ObjectMeta.Annotations["clowder/pre-hook-generation"])
	if err != nil {
		// failed to parse generation, err on the "true" side.
		return true
	}

	return preHookGen < int(app.ObjectMeta.Generation)
}

func preHookDone(app *crd.ClowdApp) bool {
	return app.Spec.PreHookJob.Name != "" && app.Annotations["clowder/pre-hook-generation"] == "done"
}
