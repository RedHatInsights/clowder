package makers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"fmt"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

//DependencyMaker makes the DependencyConfig object
type DependencyMaker struct {
	*Maker
	config config.DependenciesConfig
}

//Make function for the DependencyMaker
func (cm *DependencyMaker) Make() (ctrl.Result, error) {
	cm.config = config.DependenciesConfig{}

	// Return if no deps

	deps := cm.App.Spec.Dependencies

	if deps == nil || len(deps) == 0 {
		return ctrl.Result{}, nil
	}

	// Get all InsightsApps

	apps := crd.InsightsAppList{}
	err := cm.Client.List(cm.Ctx, &apps)

	if err != nil {
		return ctrl.Result{}, err
	}

	appMap := map[string]crd.InsightsApp{}

	for _, app := range apps.Items {
		appMap[app.Name] = app
	}

	// Iterate over all deps

	missingDeps := []string{}

	for _, dep := range cm.App.Spec.Dependencies {
		depApp, exists := appMap[dep]

		if !exists {
			missingDeps = append(missingDeps, dep)
			continue
		}

		// If app has public endpoint, add it to app config

		svcName := types.NamespacedName{
			Name:      depApp.Name,
			Namespace: depApp.Namespace,
		}

		svc := core.Service{}
		err = cm.Client.Get(cm.Ctx, svcName, &svc)

		if err != nil {
			return ctrl.Result{}, err
		}

		for _, port := range svc.Spec.Ports {
			if port.Name == "web" {
				cm.config = append(cm.config, config.DependencyConfig{
					Hostname: fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace),
					Port:     int(port.Port),
					Name:     depApp.Name,
				})
			}
		}
	}

	if len(missingDeps) > 0 {
		// TODO: Emit event
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

//ApplyConfig for the KafkaMaker
func (cm *DependencyMaker) ApplyConfig(c *config.AppConfig) {
	c.Dependencies = cm.config
}
