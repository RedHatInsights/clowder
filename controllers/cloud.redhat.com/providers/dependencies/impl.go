package dependencies

import (
	"fmt"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
)

func (dep *dependenciesProvider) makeDependencies(app *crd.ClowdApp) error {

	if dep.Provider.Env.Spec.Providers.Web.PrivatePort == 0 {
		dep.Provider.Env.Spec.Providers.Web.PrivatePort = 10000
	}

	depConfig := []config.DependencyEndpoint{}
	privDepConfig := []config.PrivateDependencyEndpoint{}

	// Process self endpoints
	appMap := map[string]crd.ClowdApp{app.Name: *app}
	appRefMap := map[string]crd.ClowdAppRef{} // empty since we're only processing self
	_ = processAppAndAppRefEndpoints(
		appMap,
		appRefMap,
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

	// Get all ClowdAppRefs

	appRefs, err := dep.getAppRefsInEnv()

	if err != nil {
		return errors.Wrap("Failed to list app refs", err)
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
		appRefs,
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

func (dep *dependenciesProvider) getAppRefsInEnv() (*crd.ClowdAppRefList, error) {
	appRefList := &crd.ClowdAppRefList{}

	err := dep.Client.List(dep.Ctx, appRefList, client.MatchingFields{"spec.envName": dep.Env.Name})
	if err != nil {
		return nil, err
	}

	return appRefList, nil
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
	appRefs *crd.ClowdAppRefList,
) (missingDeps []string) {

	appMap := map[string]crd.ClowdApp{}
	appRefMap := map[string]crd.ClowdAppRef{}

	for i := range apps.Items {
		iapp := &apps.Items[i]
		appMap[iapp.Name] = *iapp
	}

	for i := range appRefs.Items {
		iappRef := &appRefs.Items[i]
		appRefMap[iappRef.Name] = *iappRef
	}

	missingDeps = processAppAndAppRefEndpoints(appMap, appRefMap, app.Spec.Dependencies, depConfig, privDepConfig, webPort, tlsPort, privatePort, tlsPrivatePort)
	_ = processAppAndAppRefEndpoints(appMap, appRefMap, app.Spec.OptionalDependencies, depConfig, privDepConfig, webPort, tlsPort, privatePort, tlsPrivatePort)

	return missingDeps
}

func processAppAndAppRefEndpoints(
	appMap map[string]crd.ClowdApp,
	appRefMap map[string]crd.ClowdAppRef,
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
		foundInApp := false
		foundInAppRef := false

		// Check if dependency exists in ClowdApp
		if depApp, exists := appMap[dep]; exists {
			foundInApp = true
			// Process ClowdApp endpoints
			for i := range depApp.Spec.Deployments {
				// avoid implicit memory aliasing by using indexing
				innerDeployment := &depApp.Spec.Deployments[i]

				apiPaths := provutils.GetAPIPaths(innerDeployment, depApp.GetDeploymentNamespacedName(innerDeployment).Name)

				if bool(innerDeployment.Web) || innerDeployment.WebServices.Public.Enabled {
					name := depApp.GetDeploymentNamespacedName(innerDeployment).Name
					*depConfig = append(*depConfig, config.DependencyEndpoint{
						Hostname: fmt.Sprintf("%s.%s.svc", name, depApp.Namespace),
						Port:     int(webPort),
						Name:     innerDeployment.Name,
						App:      depApp.Name,
						TLSPort:  utils.IntPtr(int(tlsPort)),
						// if app has multiple paths set, set apiPath to first name for backward compatibility
						ApiPath:  apiPaths[0],
						ApiPaths: apiPaths,
					})
				}
				if innerDeployment.WebServices.Private.Enabled {
					name := depApp.GetDeploymentNamespacedName(innerDeployment).Name
					*privDepConfig = append(*privDepConfig, config.PrivateDependencyEndpoint{
						Hostname: fmt.Sprintf("%s.%s.svc", name, depApp.Namespace),
						Port:     int(privatePort),
						Name:     innerDeployment.Name,
						App:      depApp.Name,
						TLSPort:  utils.IntPtr(int(tlsPrivatePort)),
					})
				}
			}
		}

		// Check if dependency exists in ClowdAppRef
		if depAppRef, exists := appRefMap[dep]; exists {
			foundInAppRef = true
			// Process ClowdAppRef endpoints
			for i := range depAppRef.Spec.Deployments {
				// avoid implicit memory aliasing by using indexing
				innerDeployment := &depAppRef.Spec.Deployments[i]

				// Get API paths for ClowdAppRef deployment
				apiPaths := getAppRefAPIPaths(innerDeployment)

				// Use the configured port from the deployment, or fall back to defaults
				deploymentPort := webPort
				if innerDeployment.Port != 0 {
					deploymentPort = innerDeployment.Port
				}

				deploymentTLSPort := tlsPort
				if innerDeployment.TLSPort != 0 {
					deploymentTLSPort = innerDeployment.TLSPort
				}

				deploymentPrivatePort := privatePort
				if innerDeployment.PrivatePort != 0 {
					deploymentPrivatePort = innerDeployment.PrivatePort
				}

				deploymentTLSPrivatePort := tlsPrivatePort
				if innerDeployment.TLSPrivatePort != 0 {
					deploymentTLSPrivatePort = innerDeployment.TLSPrivatePort
				}

				if innerDeployment.Web || innerDeployment.WebServices.Public.Enabled {
					*depConfig = append(*depConfig, config.DependencyEndpoint{
						Hostname: innerDeployment.Hostname,
						Port:     int(deploymentPort),
						Name:     innerDeployment.Name,
						App:      depAppRef.Name,
						TLSPort:  utils.IntPtr(int(deploymentTLSPort)),
						// if app has multiple paths set, set apiPath to first name for backward compatibility
						ApiPath:  apiPaths[0],
						ApiPaths: apiPaths,
					})
				}
				if innerDeployment.WebServices.Private.Enabled {
					*privDepConfig = append(*privDepConfig, config.PrivateDependencyEndpoint{
						Hostname: innerDeployment.Hostname,
						Port:     int(deploymentPrivatePort),
						Name:     innerDeployment.Name,
						App:      depAppRef.Name,
						TLSPort:  utils.IntPtr(int(deploymentTLSPrivatePort)),
					})
				}
			}
		}

		// If not found in either, mark as missing
		if !foundInApp && !foundInAppRef {
			missingDeps = append(missingDeps, dep)
		}
	}

	return missingDeps
}

func getAppRefAPIPaths(deployment *crd.ClowdAppRefDeployment) []string {
	var apiPaths []string

	if len(deployment.APIPaths) > 0 {
		apiPaths = deployment.APIPaths
	} else {
		// Default empty API path for backward compatibility
		apiPaths = []string{""}
	}

	return apiPaths
}
