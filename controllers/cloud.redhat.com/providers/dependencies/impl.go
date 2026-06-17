package dependencies

import (
	"fmt"
	"net"
	"net/url"

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

// isInServesList checks if an app name is in the serves list
func isInServesList(appName string, serves []string) bool {
	for _, name := range serves {
		if name == appName {
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
		app.Name,
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

	// Populate V2 endpoint structures - build from CRD sources to preserve isClowdAppRef info
	allDeps := append(deps, odeps...)
	dep.makeV2DependencyEndpoints(apps, appRefs, allDeps, app.Name)
	dep.makeV2PrivateDependencyEndpoints(apps, appRefs, allDeps, app.Name)

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

	missingDeps = processAppAndAppRefEndpoints(appMap, appRefMap, app.Spec.Dependencies, depConfig, privDepConfig, envWebConfig, app.Name)
	_ = processAppAndAppRefEndpoints(appMap, appRefMap, app.Spec.OptionalDependencies, depConfig, privDepConfig, envWebConfig, app.Name)

	return missingDeps
}

// getCAPathForDependency determines the CA path based on dependency type
// Returns:
//   - *string pointing to "/cdapp/certs/service-ca.crt" for ClowdApp with TLS
//   - nil for ClowdAppRef (uses system trust store) or plaintext (no TLS)
func getCAPathForDependency(isClowdAppRef bool, tlsEnabled bool) *string {
	// Only use service-ca for in-cluster ClowdApp with TLS
	if !isClowdAppRef && tlsEnabled {
		return provutils.GetServiceCACertPath()
	}
	// All other cases: ClowdAppRef (system trust) or plaintext (no TLS)
	return nil
}

func appendPublicDependencyEndpoint(deploymentName string, deploymentWeb crd.WebDeprecated, deploymentWebServices crd.WebServices, hostname string, appName string, envWebConfig *crd.WebConfig, apiPaths []string, depConfig *[]config.DependencyEndpoint, isClowdAppRef bool) {
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
			dependencyEndpoint.TlsCAPath = getCAPathForDependency(isClowdAppRef, publicTLSEnabled)
		}

		*depConfig = append(*depConfig, dependencyEndpoint)
	}
}

func appendPrivateDependencyEndpoint(deploymentName string, deploymentWebServices crd.WebServices, hostname string, appName string, envWebConfig *crd.WebConfig, privDepConfig *[]config.PrivateDependencyEndpoint, isClowdAppRef bool) {
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
			dependencyEndpoint.TlsCAPath = getCAPathForDependency(isClowdAppRef, privateTLSEnabled)
		}

		*privDepConfig = append(*privDepConfig, dependencyEndpoint)
	}
}

func configureAppDependencyEndpoints(innerDeployment *crd.Deployment, depApp crd.ClowdApp, depConfig *[]config.DependencyEndpoint, privDepConfig *[]config.PrivateDependencyEndpoint, envWebConfig *crd.WebConfig) {
	serviceName := depApp.GetDeploymentNamespacedName(innerDeployment).Name
	apiPaths := provutils.GetAPIPaths(innerDeployment, serviceName)

	// For ClowdApp, construct the internal Kubernetes service hostname
	hostname := fmt.Sprintf("%s.%s.svc", serviceName, depApp.Namespace)

	appendPublicDependencyEndpoint(innerDeployment.Name, innerDeployment.Web, innerDeployment.WebServices, hostname, depApp.Name, envWebConfig, apiPaths, depConfig, false)
	appendPrivateDependencyEndpoint(innerDeployment.Name, innerDeployment.WebServices, hostname, depApp.Name, envWebConfig, privDepConfig, false)
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

	// For ClowdAppRef TLS, if PrivatePort is not set but Port is, use Port as the default for PrivatePort
	// This allows ClowdAppRef to specify only one TLS port that serves both public and private endpoints
	tlsPrivatePort := coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.PrivatePort, envWebConfig.TLS.PrivatePort)
	if depAppRef.Spec.RemoteEnvironment.TLS.Enabled && tlsPrivatePort == 0 {
		tlsPrivatePort = coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.Port, envWebConfig.TLS.Port)
	}

	webConfig := crd.WebConfig{
		Port:           coalesceInt32(depAppRef.Spec.RemoteEnvironment.Port, envWebConfig.Port),
		PrivatePort:    coalesceInt32(depAppRef.Spec.RemoteEnvironment.PrivatePort, envWebConfig.PrivatePort),
		H2CPort:        coalesceInt32(depAppRef.Spec.RemoteEnvironment.H2CPort, envWebConfig.H2CPort),
		H2CPrivatePort: coalesceInt32(depAppRef.Spec.RemoteEnvironment.H2CPrivatePort, envWebConfig.H2CPrivatePort),
		TLS: crd.TLS{
			Enabled:        depAppRef.Spec.RemoteEnvironment.TLS.Enabled || envWebConfig.TLS.Enabled,
			Port:           coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.Port, envWebConfig.TLS.Port),
			H2CPort:        coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.H2CPort, envWebConfig.TLS.H2CPort),
			PrivatePort:    tlsPrivatePort,
			H2CPrivatePort: coalesceInt32(depAppRef.Spec.RemoteEnvironment.TLS.H2CPrivatePort, envWebConfig.TLS.H2CPrivatePort),
		},
	}

	// For ClowdAppRef, use the explicit hostname from the deployment spec instead of generating one
	appendPublicDependencyEndpoint(innerDeployment.Name, innerDeployment.Web, innerDeployment.WebServices, innerDeployment.Hostname, depAppRef.Name, &webConfig, apiPaths, depConfig, true)
	appendPrivateDependencyEndpoint(innerDeployment.Name, innerDeployment.WebServices, innerDeployment.Hostname, depAppRef.Name, &webConfig, privDepConfig, true)
}

func processAppAndAppRefEndpoints(
	appMap map[string]crd.ClowdApp,
	appRefMap map[string]crd.ClowdAppRef,
	depList []string,
	depConfig *[]config.DependencyEndpoint,
	privDepConfig *[]config.PrivateDependencyEndpoint,
	envWebConfig *crd.WebConfig,
	consumerAppName string,
) (missingDeps []string) {

	missingDeps = []string{}

	for _, dep := range depList {
		depApp, hasApp := appMap[dep]
		depAppRef, hasAppRef := appRefMap[dep]

		useAppRef := false

		if hasApp && hasAppRef {
			// Both ClowdApp and ClowdAppRef exist - check serves field
			useAppRef = isInServesList(consumerAppName, depAppRef.Spec.Serves)
		} else if hasAppRef {
			// Only ClowdAppRef exists
			useAppRef = true
		}
		// else: only ClowdApp exists or neither exists

		switch {
		case useAppRef && hasAppRef:
			// Use ClowdAppRef endpoints
			for i := range depAppRef.Spec.Deployments {
				// avoid implicit memory aliasing by using indexing
				innerDeployment := &depAppRef.Spec.Deployments[i]
				configureAppRefDependencyEndpoints(innerDeployment, depAppRef, depConfig, privDepConfig, envWebConfig)
			}
		case hasApp:
			// Use ClowdApp endpoints
			for i := range depApp.Spec.Deployments {
				// avoid implicit memory aliasing by using indexing
				innerDeployment := &depApp.Spec.Deployments[i]
				configureAppDependencyEndpoints(innerDeployment, depApp, depConfig, privDepConfig, envWebConfig)
			}
		default:
			// Neither ClowdApp nor ClowdAppRef exists
			missingDeps = append(missingDeps, dep)
		}
	}

	return missingDeps
}

// constructEndpointURI builds a complete URI from protocol, hostname, and port
func constructEndpointURI(protocol string, hostname string, port int) string {
	u := &url.URL{
		Scheme: protocol,
		Host:   net.JoinHostPort(hostname, fmt.Sprintf("%d", port)),
	}
	return u.String()
}

// endpointPortInfo holds port configuration for building a V2 endpoint
type endpointPortInfo struct {
	port       int
	tlsPort    int
	h2cPort    int
	h2cTLSPort int
}

// extractPublicPorts extracts public port configuration from WebServices and WebConfig
func extractPublicPorts(webServices *crd.WebServices, web crd.WebDeprecated, envWebConfig *crd.WebConfig) endpointPortInfo {
	info := endpointPortInfo{}

	if bool(web) || webServices.Public.Enabled {
		info.port = int(envWebConfig.Port)
		if provutils.IsPublicTLSEnabled(webServices, &envWebConfig.TLS) {
			info.tlsPort = int(envWebConfig.TLS.Port)
		}
	}

	if webServices.Public.H2CEnabled {
		if envWebConfig.H2CPort != 0 {
			info.h2cPort = int(envWebConfig.H2CPort)
		}
		if provutils.IsPublicTLSEnabled(webServices, &envWebConfig.TLS) && envWebConfig.TLS.H2CPort != 0 {
			info.h2cTLSPort = int(envWebConfig.TLS.H2CPort)
		}
	}

	return info
}

// extractPrivatePorts extracts private port configuration from WebServices and WebConfig
func extractPrivatePorts(webServices *crd.WebServices, envWebConfig *crd.WebConfig) endpointPortInfo {
	info := endpointPortInfo{}

	if webServices.Private.Enabled {
		info.port = int(envWebConfig.PrivatePort)
		if info.port == 0 {
			info.port = 10000
		}
		if provutils.IsPrivateTLSEnabled(webServices, &envWebConfig.TLS) {
			info.tlsPort = int(envWebConfig.TLS.PrivatePort)
		}
	}

	if webServices.Private.H2CEnabled {
		if envWebConfig.H2CPrivatePort != 0 {
			info.h2cPort = int(envWebConfig.H2CPrivatePort)
		}
		if provutils.IsPrivateTLSEnabled(webServices, &envWebConfig.TLS) && envWebConfig.TLS.H2CPrivatePort != 0 {
			info.h2cTLSPort = int(envWebConfig.TLS.H2CPrivatePort)
		}
	}

	return info
}

// buildV2EndpointFromPorts constructs a V2 endpoint from port info
// Returns nil if no valid port is configured
func buildV2EndpointFromPorts(hostname string, ports endpointPortInfo, authenticated bool, isClowdAppRef bool) *config.DependencyEndpointV2 {
	var uri string
	var tlsCAPath *string

	// Priority: TLS > H2C TLS > H2C > plaintext
	switch {
	case ports.tlsPort > 0:
		uri = constructEndpointURI("https", hostname, ports.tlsPort)
		if !isClowdAppRef {
			tlsCAPath = provutils.GetServiceCACertPath()
		}
	case ports.h2cTLSPort > 0:
		uri = constructEndpointURI("https", hostname, ports.h2cTLSPort)
		if !isClowdAppRef {
			tlsCAPath = provutils.GetServiceCACertPath()
		}
	case ports.h2cPort > 0:
		uri = constructEndpointURI("http", hostname, ports.h2cPort)
	case ports.port > 0:
		uri = constructEndpointURI("http", hostname, ports.port)
	default:
		return nil // No valid port
	}

	endpoint := config.DependencyEndpointV2{
		Uri:           uri,
		Authenticated: authenticated,
	}

	if tlsCAPath != nil {
		endpoint.CaCertificate = tlsCAPath
	}

	return &endpoint
}

// buildV2EndpointsForApp creates V2 endpoints for ClowdApp (in-cluster, authenticated: false)
func buildV2EndpointsForApp(app *crd.ClowdApp, envWebConfig *crd.WebConfig, v2Map map[string]map[string]config.DependencyEndpointV2) {
	for i := range app.Spec.Deployments {
		deployment := &app.Spec.Deployments[i]

		ports := extractPublicPorts(&deployment.WebServices, deployment.Web, envWebConfig)
		hostname := fmt.Sprintf("%s-%s.%s.svc", app.Name, deployment.Name, app.Namespace)

		endpoint := buildV2EndpointFromPorts(hostname, ports, false, false) // authenticated=false, isClowdAppRef=false
		if endpoint == nil {
			continue // No valid port configured
		}

		if _, exists := v2Map[app.Name]; !exists {
			v2Map[app.Name] = make(map[string]config.DependencyEndpointV2)
		}

		v2Map[app.Name][deployment.Name] = *endpoint
	}
}

// buildV2EndpointsForAppRef creates V2 endpoints for ClowdAppRef (cross-cluster, authenticated: true)
func buildV2EndpointsForAppRef(appRef *crd.ClowdAppRef, envWebConfig *crd.WebConfig, v2Map map[string]map[string]config.DependencyEndpointV2) {
	// Build webConfig from ClowdAppRef's remoteEnvironment + env fallback
	tlsPrivatePort := coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.PrivatePort, envWebConfig.TLS.PrivatePort)
	if appRef.Spec.RemoteEnvironment.TLS.Enabled && tlsPrivatePort == 0 {
		tlsPrivatePort = coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.Port, envWebConfig.TLS.Port)
	}

	webConfig := crd.WebConfig{
		Port:           coalesceInt32(appRef.Spec.RemoteEnvironment.Port, envWebConfig.Port),
		PrivatePort:    coalesceInt32(appRef.Spec.RemoteEnvironment.PrivatePort, envWebConfig.PrivatePort),
		H2CPort:        coalesceInt32(appRef.Spec.RemoteEnvironment.H2CPort, envWebConfig.H2CPort),
		H2CPrivatePort: coalesceInt32(appRef.Spec.RemoteEnvironment.H2CPrivatePort, envWebConfig.H2CPrivatePort),
		TLS: crd.TLS{
			Enabled:        appRef.Spec.RemoteEnvironment.TLS.Enabled,
			Port:           coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.Port, envWebConfig.TLS.Port),
			H2CPort:        coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.H2CPort, envWebConfig.TLS.H2CPort),
			PrivatePort:    tlsPrivatePort,
			H2CPrivatePort: coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.H2CPrivatePort, envWebConfig.TLS.H2CPrivatePort),
		},
	}

	for i := range appRef.Spec.Deployments {
		deployment := &appRef.Spec.Deployments[i]

		ports := extractPublicPorts(&deployment.WebServices, deployment.Web, &webConfig)
		endpoint := buildV2EndpointFromPorts(deployment.Hostname, ports, true, true) // authenticated=true, isClowdAppRef=true
		if endpoint == nil {
			continue
		}

		if _, exists := v2Map[appRef.Name]; !exists {
			v2Map[appRef.Name] = make(map[string]config.DependencyEndpointV2)
		}

		v2Map[appRef.Name][deployment.Name] = *endpoint
	}
}

// buildV2EndpointMap transforms V1 endpoint data into V2 format
// Returns map[appName]map[serviceName]DependencyEndpointV2
func buildV2EndpointMap(endpoints []config.DependencyEndpoint) map[string]map[string]config.DependencyEndpointV2 {
	if len(endpoints) == 0 {
		return nil
	}

	v2Map := make(map[string]map[string]config.DependencyEndpointV2)

	for _, ep := range endpoints {
		// Initialize app map if it doesn't exist
		if _, exists := v2Map[ep.App]; !exists {
			v2Map[ep.App] = make(map[string]config.DependencyEndpointV2)
		}

		// Determine the single correct endpoint to expose
		// Priority: TLS > H2C TLS > H2C > plaintext
		var uri string
		var tlsCAPath *string

		switch {
		case ep.TlsPort != nil && *ep.TlsPort > 0:
			// Use HTTPS
			uri = constructEndpointURI("https", ep.Hostname, *ep.TlsPort)
			tlsCAPath = ep.TlsCAPath
		case ep.H2CTLSPort != nil && *ep.H2CTLSPort > 0:
			// Use H2C with TLS
			uri = constructEndpointURI("https", ep.Hostname, *ep.H2CTLSPort)
			tlsCAPath = ep.TlsCAPath
		case ep.H2CPort != nil && *ep.H2CPort > 0:
			// Use H2C plaintext
			uri = constructEndpointURI("http", ep.Hostname, *ep.H2CPort)
		case ep.Port > 0:
			// Use HTTP plaintext
			uri = constructEndpointURI("http", ep.Hostname, ep.Port)
		default:
			// No valid port configured, skip
			continue
		}

		endpoint := config.DependencyEndpointV2{
			Uri:           uri,
			Authenticated: false, // Default to false - this function doesn't have ClowdApp/ClowdAppRef context
		}

		// Only include ca_certificate field if TLS is used AND CA is needed
		if tlsCAPath != nil {
			endpoint.CaCertificate = tlsCAPath
		}

		v2Map[ep.App][ep.Name] = endpoint
	}

	return v2Map
}

// makeV2DependencyEndpoints populates the V2 public dependency endpoints
// Builds directly from ClowdApp/ClowdAppRef sources to preserve isClowdAppRef information
func (dep *dependenciesProvider) makeV2DependencyEndpoints(apps *crd.ClowdAppList, appRefs *crd.ClowdAppRefList, depList []string, consumerAppName string) {
	if len(depList) == 0 {
		return
	}

	v2Map := make(map[string]map[string]config.DependencyEndpointV2)
	envWebConfig := &dep.Env.Spec.Providers.Web

	// Build maps for lookups
	appMap := make(map[string]crd.ClowdApp)
	for i := range apps.Items {
		iapp := &apps.Items[i]
		appMap[iapp.Name] = *iapp
	}

	appRefMap := make(map[string]crd.ClowdAppRef)
	for i := range appRefs.Items {
		iappRef := &appRefs.Items[i]
		appRefMap[iappRef.Name] = *iappRef
	}

	// Process each dependency
	for _, depName := range depList {
		depApp, hasApp := appMap[depName]
		depAppRef, hasAppRef := appRefMap[depName]

		useAppRef := false
		if hasApp && hasAppRef {
			useAppRef = isInServesList(consumerAppName, depAppRef.Spec.Serves)
		} else if hasAppRef {
			useAppRef = true
		}

		if useAppRef && hasAppRef {
			// ClowdAppRef - authenticated: true
			buildV2EndpointsForAppRef(&depAppRef, envWebConfig, v2Map)
		} else if hasApp {
			// ClowdApp - authenticated: false
			buildV2EndpointsForApp(&depApp, envWebConfig, v2Map)
		}
	}

	if len(v2Map) > 0 {
		v2Interface := make(config.AppConfigDependencyEndpointsV2)
		for appName, services := range v2Map {
			v2Interface[appName] = services
		}

		dep.Config.DependencyEndpoints = &config.AppConfigDependencyEndpoints{
			V2: v2Interface,
		}
	}
}

// makeV2PrivateDependencyEndpoints populates the V2 private dependency endpoints
// Builds directly from ClowdApp/ClowdAppRef sources to preserve isClowdAppRef information
func (dep *dependenciesProvider) makeV2PrivateDependencyEndpoints(apps *crd.ClowdAppList, appRefs *crd.ClowdAppRefList, depList []string, consumerAppName string) {
	if len(depList) == 0 {
		return
	}

	v2Map := make(map[string]map[string]config.DependencyEndpointV2)
	envWebConfig := &dep.Env.Spec.Providers.Web

	// Build maps for lookups
	appMap := make(map[string]crd.ClowdApp)
	for i := range apps.Items {
		iapp := &apps.Items[i]
		appMap[iapp.Name] = *iapp
	}

	appRefMap := make(map[string]crd.ClowdAppRef)
	for i := range appRefs.Items {
		iappRef := &appRefs.Items[i]
		appRefMap[iappRef.Name] = *iappRef
	}

	// Process each dependency
	for _, depName := range depList {
		depApp, hasApp := appMap[depName]
		depAppRef, hasAppRef := appRefMap[depName]

		useAppRef := false
		if hasApp && hasAppRef {
			useAppRef = isInServesList(consumerAppName, depAppRef.Spec.Serves)
		} else if hasAppRef {
			useAppRef = true
		}

		if useAppRef && hasAppRef {
			// ClowdAppRef - authenticated: true
			buildV2PrivateEndpointsForAppRef(&depAppRef, envWebConfig, v2Map)
		} else if hasApp {
			// ClowdApp - authenticated: false
			buildV2PrivateEndpointsForApp(&depApp, envWebConfig, v2Map)
		}
	}

	if len(v2Map) > 0 {
		v2Interface := make(config.AppConfigPrivateDependencyEndpointsV2)
		for appName, services := range v2Map {
			v2Interface[appName] = services
		}

		dep.Config.PrivateDependencyEndpoints = &config.AppConfigPrivateDependencyEndpoints{
			V2: v2Interface,
		}
	}
}

// buildV2PrivateEndpointsForApp creates V2 private endpoints for ClowdApp (in-cluster, authenticated: false)
func buildV2PrivateEndpointsForApp(app *crd.ClowdApp, envWebConfig *crd.WebConfig, v2Map map[string]map[string]config.DependencyEndpointV2) {
	for i := range app.Spec.Deployments {
		deployment := &app.Spec.Deployments[i]

		ports := extractPrivatePorts(&deployment.WebServices, envWebConfig)
		hostname := fmt.Sprintf("%s-%s.%s.svc", app.Name, deployment.Name, app.Namespace)

		endpoint := buildV2EndpointFromPorts(hostname, ports, false, false) // authenticated=false, isClowdAppRef=false
		if endpoint == nil {
			continue
		}

		if _, exists := v2Map[app.Name]; !exists {
			v2Map[app.Name] = make(map[string]config.DependencyEndpointV2)
		}

		v2Map[app.Name][deployment.Name] = *endpoint
	}
}

// buildV2PrivateEndpointsForAppRef creates V2 private endpoints for ClowdAppRef (cross-cluster, authenticated: true)
func buildV2PrivateEndpointsForAppRef(appRef *crd.ClowdAppRef, envWebConfig *crd.WebConfig, v2Map map[string]map[string]config.DependencyEndpointV2) {
	// Build webConfig from ClowdAppRef's remoteEnvironment + env fallback
	tlsPrivatePort := coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.PrivatePort, envWebConfig.TLS.PrivatePort)
	if appRef.Spec.RemoteEnvironment.TLS.Enabled && tlsPrivatePort == 0 {
		tlsPrivatePort = coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.Port, envWebConfig.TLS.Port)
	}

	webConfig := crd.WebConfig{
		Port:           coalesceInt32(appRef.Spec.RemoteEnvironment.Port, envWebConfig.Port),
		PrivatePort:    coalesceInt32(appRef.Spec.RemoteEnvironment.PrivatePort, envWebConfig.PrivatePort),
		H2CPort:        coalesceInt32(appRef.Spec.RemoteEnvironment.H2CPort, envWebConfig.H2CPort),
		H2CPrivatePort: coalesceInt32(appRef.Spec.RemoteEnvironment.H2CPrivatePort, envWebConfig.H2CPrivatePort),
		TLS: crd.TLS{
			Enabled:        appRef.Spec.RemoteEnvironment.TLS.Enabled,
			Port:           coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.Port, envWebConfig.TLS.Port),
			H2CPort:        coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.H2CPort, envWebConfig.TLS.H2CPort),
			PrivatePort:    tlsPrivatePort,
			H2CPrivatePort: coalesceInt32(appRef.Spec.RemoteEnvironment.TLS.H2CPrivatePort, envWebConfig.TLS.H2CPrivatePort),
		},
	}

	for i := range appRef.Spec.Deployments {
		deployment := &appRef.Spec.Deployments[i]

		ports := extractPrivatePorts(&deployment.WebServices, &webConfig)
		endpoint := buildV2EndpointFromPorts(deployment.Hostname, ports, true, true) // authenticated=true, isClowdAppRef=true
		if endpoint == nil {
			continue
		}

		if _, exists := v2Map[appRef.Name]; !exists {
			v2Map[appRef.Name] = make(map[string]config.DependencyEndpointV2)
		}

		v2Map[appRef.Name][deployment.Name] = *endpoint
	}
}
