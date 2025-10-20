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

func anyNotZero(ints ...int) bool {
	for _, i := range ints {
		if i != 0 {
			return true
		}
	}
	return false
}

func (dep *dependenciesProvider) makeDependencies(app *crd.ClowdApp) error {

	if dep.Env.Spec.Providers.Web.PrivatePort == 0 {
		dep.Env.Spec.Providers.Web.PrivatePort = 10000
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
		&dep.Env.Spec.Providers.Web,
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
		&dep.Env.Spec.Providers.Web,
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
	envWebConfig *crd.WebConfig,
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

	missingDeps = processAppAndAppRefEndpoints(appMap, appRefMap, app.Spec.Dependencies, depConfig, privDepConfig, envWebConfig)
	_ = processAppAndAppRefEndpoints(appMap, appRefMap, app.Spec.OptionalDependencies, depConfig, privDepConfig, envWebConfig)

	return missingDeps
}

func appendPublicDependencyEndpoint(deploymentName string, deploymentWeb crd.WebDeprecated, deploymentWebServices crd.WebServices, hostname string, appName string, envWebConfig *crd.WebConfig, apiPaths []string, depConfig *[]config.DependencyEndpoint) {
	port := int(0)
	tlsPort := int(0)
	h2cPort := int(0)
	h2cTLSPort := int(0)

	var publicTLSEnabled bool

	if bool(deploymentWeb) || deploymentWebServices.Public.Enabled {
		port = int(envWebConfig.Port)
		if provutils.IsPublicTLSEnabled(&deploymentWebServices, &envWebConfig.TLS) {
			tlsPort = int(envWebConfig.TLS.Port)
			publicTLSEnabled = true
		}
	}

	if deploymentWebServices.Public.H2CEnabled {
		if envWebConfig.H2CPort != 0 {
			h2cPort = int(envWebConfig.H2CPort)
		}
		if provutils.IsPublicTLSEnabled(&deploymentWebServices, &envWebConfig.TLS) && envWebConfig.TLS.H2CPort != 0 {
			h2cTLSPort = int(envWebConfig.TLS.H2CPort)
			publicTLSEnabled = true
		}
	}

	if anyNotZero(port, tlsPort, h2cPort, h2cTLSPort) {
		dependencyEndpoint := config.DependencyEndpoint{
			Hostname:   hostname,
			Port:       port,
			Name:       deploymentName,
			App:        appName,
			TlsPort:    utils.IntPtr(tlsPort),
			H2CPort:    utils.IntPtr(h2cPort),
			H2CTLSPort: utils.IntPtr(h2cTLSPort),
			// if app has multiple paths set, set apiPath to first name for backward compatibility
			ApiPath:  apiPaths[0],
			ApiPaths: apiPaths,
		}

		if publicTLSEnabled {
			dependencyEndpoint.TlsCAPath = provutils.GetServiceCACertPath()
		}

		*depConfig = append(*depConfig, dependencyEndpoint)
	}
}

func appendPrivateDependencyEndpoint(deploymentName string, deploymentWebServices crd.WebServices, hostname string, appName string, envWebConfig *crd.WebConfig, privDepConfig *[]config.PrivateDependencyEndpoint) {
	privatePort := int(0)
	tlsPrivatePort := int(0)
	h2cPrivatePort := int(0)
	h2cTLSPrivatePort := int(0)

	var privateTLSEnabled bool

	if deploymentWebServices.Private.Enabled {
		privatePort = int(envWebConfig.PrivatePort)
		if provutils.IsPrivateTLSEnabled(&deploymentWebServices, &envWebConfig.TLS) {
			tlsPrivatePort = int(envWebConfig.TLS.PrivatePort)
			privateTLSEnabled = true
		}
	}

	if deploymentWebServices.Private.H2CEnabled {
		if envWebConfig.H2CPrivatePort != 0 {
			h2cPrivatePort = int(envWebConfig.H2CPrivatePort)
		}
		if provutils.IsPrivateTLSEnabled(&deploymentWebServices, &envWebConfig.TLS) && envWebConfig.TLS.H2CPrivatePort != 0 {
			h2cTLSPrivatePort = int(envWebConfig.TLS.H2CPrivatePort)
			privateTLSEnabled = true
		}
	}

	if anyNotZero(privatePort, tlsPrivatePort, h2cPrivatePort, h2cTLSPrivatePort) {
		dependencyEndpoint := config.PrivateDependencyEndpoint{
			Hostname:   hostname,
			Port:       int(privatePort),
			Name:       deploymentName,
			App:        appName,
			TlsPort:    utils.IntPtr(tlsPrivatePort),
			H2CPort:    utils.IntPtr(h2cPrivatePort),
			H2CTLSPort: utils.IntPtr(h2cTLSPrivatePort),
		}

		if privateTLSEnabled {
			dependencyEndpoint.TlsCAPath = provutils.GetServiceCACertPath()
		}

		*privDepConfig = append(*privDepConfig, dependencyEndpoint)
	}
}

func configureAppDependencyEndpoints(innerDeployment *crd.Deployment, depApp crd.ClowdApp, depConfig *[]config.DependencyEndpoint, privDepConfig *[]config.PrivateDependencyEndpoint, envWebConfig *crd.WebConfig) {
	serviceName := depApp.GetDeploymentNamespacedName(innerDeployment).Name
	apiPaths := provutils.GetAPIPaths(innerDeployment, serviceName)

	// For ClowdApp, construct the internal Kubernetes service hostname
	hostname := fmt.Sprintf("%s.%s.svc", serviceName, depApp.Namespace)

	appendPublicDependencyEndpoint(innerDeployment.Name, innerDeployment.Web, innerDeployment.WebServices, hostname, depApp.Name, envWebConfig, apiPaths, depConfig)
	appendPrivateDependencyEndpoint(innerDeployment.Name, innerDeployment.WebServices, hostname, depApp.Name, envWebConfig, privDepConfig)
}

func coalesceInt32(vals ...int32) int32 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

func configureAppRefDependencyEndpoints(innerDeployment *crd.ClowdAppRefDeployment, depAppRef crd.ClowdAppRef, depConfig *[]config.DependencyEndpoint, privDepConfig *[]config.PrivateDependencyEndpoint, envWebConfig *crd.WebConfig) {
	serviceName := depAppRef.GetDeploymentNamespacedName(innerDeployment).Name
	apiPaths := provutils.GetAPIPaths(innerDeployment, serviceName)

	webConfig := crd.WebConfig{
		Port:           coalesceInt32(depAppRef.Spec.RemoteEnvironment.Port, envWebConfig.Port),
		PrivatePort:    coalesceInt32(depAppRef.Spec.RemoteEnvironment.PrivatePort, envWebConfig.PrivatePort),
		H2CPort:        coalesceInt32(depAppRef.Spec.RemoteEnvironment.H2CPort, envWebConfig.H2CPort),
		H2CPrivatePort: coalesceInt32(depAppRef.Spec.RemoteEnvironment.H2CPrivatePort, envWebConfig.H2CPrivatePort),
		TLS: crd.TLS{
			Port:           coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.Port, envWebConfig.TLS.Port),
			H2CPort:        coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.H2CPort, envWebConfig.TLS.H2CPort),
			PrivatePort:    coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.PrivatePort, envWebConfig.TLS.PrivatePort),
			H2CPrivatePort: coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.H2CPrivatePort, envWebConfig.TLS.H2CPrivatePort),
		},
	}

	// For ClowdAppRef, use the explicit hostname from the deployment spec instead of generating one
	appendPublicDependencyEndpoint(innerDeployment.Name, innerDeployment.Web, innerDeployment.WebServices, innerDeployment.Hostname, depAppRef.Name, &webConfig, apiPaths, depConfig)
	appendPrivateDependencyEndpoint(innerDeployment.Name, innerDeployment.WebServices, innerDeployment.Hostname, depAppRef.Name, &webConfig, privDepConfig)
}

func processAppAndAppRefEndpoints(
	appMap map[string]crd.ClowdApp,
	appRefMap map[string]crd.ClowdAppRef,
	depList []string,
	depConfig *[]config.DependencyEndpoint,
	privDepConfig *[]config.PrivateDependencyEndpoint,
	envWebConfig *crd.WebConfig,
) (missingDeps []string) {

	missingDeps = []string{}

	for _, dep := range depList {
		if depApp, exists := appMap[dep]; exists {
			// If dependency exists in ClowdApp, configure endpoints for each deployment
			for i := range depApp.Spec.Deployments {
				// avoid implicit memory aliasing by using indexing
				innerDeployment := &depApp.Spec.Deployments[i]
				configureAppDependencyEndpoints(innerDeployment, depApp, depConfig, privDepConfig, envWebConfig)
			}
		} else if depAppRef, exists := appRefMap[dep]; exists {
			// If dependency exists in ClowdAppRef, configure endpoints for each deployment
			for i := range depAppRef.Spec.Deployments {
				// avoid implicit memory aliasing by using indexing
				innerDeployment := &depAppRef.Spec.Deployments[i]
				configureAppRefDependencyEndpoints(innerDeployment, depAppRef, depConfig, privDepConfig, envWebConfig)
			}
		} else {
			// If dependency is not found in ClowdApp or ClowdAppRef, mark as missing
			missingDeps = append(missingDeps, dep)
		}
	}

	return missingDeps
}
