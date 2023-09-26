package dependencies

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

func (dep *dependenciesProvider) makeDependencies(app *crd.ClowdApp) error {

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
		dep.Provider.Env.Spec.Providers.Web.TLS.Port,
		dep.Provider.Env.Spec.Providers.Web.PrivatePort,
		dep.Provider.Env.Spec.Providers.Web.TLS.PrivatePort,
	)

	// Return if no deps

	deps := app.Spec.Dependencies
	odeps := app.Spec.OptionalDependencies
	if len(deps) == 0 && len(odeps) == 0 {
		dep.Config.Endpoints = depConfig
		dep.Config.PrivateEndpoints = privDepConfig
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
		dep.Provider.Env.Spec.Providers.Web.TLS.Port,
		dep.Provider.Env.Spec.Providers.Web.PrivatePort,
		dep.Provider.Env.Spec.Providers.Web.TLS.PrivatePort,
		app,
		apps,
	)

	if len(missingDeps) > 0 {
		missingDepStructs := []errors.MissingDependency{}
		for _, dep := range missingDeps {
			missingDepStructs = append(missingDepStructs, errors.MissingDependency{
				Source:  "service",
				Details: dep,
			})
		}
		return &errors.MissingDependencies{MissingDeps: missingDepStructs}
	}

	dep.Config.Endpoints = depConfig
	dep.Config.PrivateEndpoints = privDepConfig
	return nil
}

func makeDepConfig(
	depConfig *[]config.DependencyEndpoint,
	privDepConfig *[]config.PrivateDependencyEndpoint,
	webPort int32,
	tlsPort int32,
	privatePort int32,
	tlsPrivatePort int32,
	app *crd.ClowdApp,
	apps *crd.ClowdAppList,
) (missingDeps []string) {

	appMap := map[string]crd.ClowdApp{}

	for _, iapp := range apps.Items {
		appMap[iapp.Name] = iapp
	}

	missingDeps = processAppEndpoints(appMap, app.Spec.Dependencies, depConfig, privDepConfig, webPort, tlsPort, privatePort, tlsPrivatePort)
	_ = processAppEndpoints(appMap, app.Spec.OptionalDependencies, depConfig, privDepConfig, webPort, tlsPort, privatePort, tlsPrivatePort)

	return missingDeps
}

func processAppEndpoints(
	appMap map[string]crd.ClowdApp,
	depList []string,
	depConfig *[]config.DependencyEndpoint,
	privDepConfig *[]config.PrivateDependencyEndpoint,
	webPort int32,
	tlsPort int32,
	privatePort int32,
	tlsPrivatePort int32,
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
			// avoid implicit memory aliasing
			innerDeployment := deployment

			apiPaths := provutils.GetAPIPaths(&innerDeployment, depApp.GetDeploymentNamespacedName(&innerDeployment).Name)

			if bool(innerDeployment.Web) || innerDeployment.WebServices.Public.Enabled {
				name := depApp.GetDeploymentNamespacedName(&innerDeployment).Name
				*depConfig = append(*depConfig, config.DependencyEndpoint{
					Hostname: fmt.Sprintf("%s.%s.svc", name, depApp.Namespace),
					Port:     int(webPort),
					Name:     innerDeployment.Name,
					App:      depApp.Name,
					TlsPort:  utils.IntPtr(int(tlsPort)),
					// if app has multiple paths set, set apiPath to first name for backward compatibility
					ApiPath:  apiPaths[0],
					ApiPaths: apiPaths,
				})
			}
			if innerDeployment.WebServices.Private.Enabled {
				name := depApp.GetDeploymentNamespacedName(&innerDeployment).Name
				*privDepConfig = append(*privDepConfig, config.PrivateDependencyEndpoint{
					Hostname: fmt.Sprintf("%s.%s.svc", name, depApp.Namespace),
					Port:     int(privatePort),
					Name:     innerDeployment.Name,
					App:      depApp.Name,
					TlsPort:  utils.IntPtr(int(tlsPrivatePort)),
				})
			}
		}
	}

	return missingDeps
}
