package dependencies

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
)

func (dep *dependenciesProvider) makeDependencies(app *crd.ClowdApp, c *config.AppConfig) error {

	if dep.Provider.Env.Spec.Providers.Web.PrivatePort == 0 {
		dep.Provider.Env.Spec.Providers.Web.PrivatePort = 10000
	}

	depConfig := []config.DependencyEndpoint{}
	privDepConfig := []config.PrivateDependencyEndpoint{}

	processAppEndpoints(
		map[string]crd.ClowdApp{app.Name: *app},
		[]string{app.Name},
		&depConfig,
		&privDepConfig,
		dep.Provider.Env.Spec.Providers.Web.Port,
		dep.Provider.Env.Spec.Providers.Web.PrivatePort,
	)

	// Return if no deps

	deps := app.Spec.Dependencies
	odeps := app.Spec.OptionalDependencies
	if len(deps) == 0 && len(odeps) == 0 {
		c.Endpoints = depConfig
		c.PrivateEndpoints = privDepConfig
		return nil
	}

	// Get all ClowdApps

	apps, err := dep.Env.GetAppsInEnv(dep.Ctx, dep.Client)

	if err != nil {
		return errors.Wrap("Failed to list apps", err)
	}

	// Iterate over all deps
	missingDeps := makeDepConfig(
		&depConfig,
		&privDepConfig,
		dep.Provider.Env.Spec.Providers.Web.Port,
		dep.Provider.Env.Spec.Providers.Web.PrivatePort,
		app,
		apps,
	)

	if len(missingDeps) > 0 {
		missingDepStructs := []errors.MissingDependency{}
		for _, dep := range missingDeps {
			missingDepStructs = append(missingDepStructs, errors.MissingDependency{
				Source:  "service",
				App:     app.Name,
				Details: dep,
			})
		}
		return &errors.MissingDependencies{MissingDeps: missingDepStructs}
	}

	c.Endpoints = depConfig
	c.PrivateEndpoints = privDepConfig
	return nil
}

func makeDepConfig(
	depConfig *[]config.DependencyEndpoint,
	privDepConfig *[]config.PrivateDependencyEndpoint,
	webPort int32,
	privatePort int32,
	app *crd.ClowdApp,
	apps *crd.ClowdAppList,
) (missingDeps []string) {

	appMap := map[string]crd.ClowdApp{}

	for _, iapp := range apps.Items {
		appMap[iapp.Name] = iapp
	}

	missingDeps = processAppEndpoints(appMap, app.Spec.Dependencies, depConfig, privDepConfig, webPort, privatePort)
	_ = processAppEndpoints(appMap, app.Spec.OptionalDependencies, depConfig, privDepConfig, webPort, privatePort)

	return missingDeps
}

func processAppEndpoints(
	appMap map[string]crd.ClowdApp,
	depList []string,
	depConfig *[]config.DependencyEndpoint,
	privDepConfig *[]config.PrivateDependencyEndpoint,
	webPort int32,
	privatePort int32,
) (missingDeps []string) {

	missingDeps = []string{}

	for _, dep := range depList {
		depApp, exists := appMap[dep]
		if !exists {
			missingDeps = append(missingDeps, dep)
			continue
		}

		// If app has public endpoint, add it to app config

		for _, deployment := range depApp.Spec.Deployments {
			if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
				name := depApp.GetDeploymentNamespacedName(&deployment).Name
				*depConfig = append(*depConfig, config.DependencyEndpoint{
					Hostname: fmt.Sprintf("%s.%s.svc", name, depApp.Namespace),
					Port:     int(webPort),
					Name:     deployment.Name,
					App:      depApp.Name,
				})
			}
			if deployment.WebServices.Private.Enabled {
				name := depApp.GetDeploymentNamespacedName(&deployment).Name
				*privDepConfig = append(*privDepConfig, config.PrivateDependencyEndpoint{
					Hostname: fmt.Sprintf("%s.%s.svc", name, depApp.Namespace),
					Port:     int(privatePort),
					Name:     deployment.Name,
					App:      depApp.Name,
				})
			}
		}
	}

	return missingDeps
}
