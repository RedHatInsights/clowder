// Package clowderconfig provides a struct for the Clowder config.
package clowderconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

// ClowderConfig is the struct for the Clowder config.
type ClowderConfig struct {
	Images struct {
		MBOP                    string `json:"mbop"`
		Caddy                   string `json:"caddy"`
		CaddyGateway            string `json:"caddyGateway"`
		CaddyProxy              string `json:"caddyProxy"`
		Keycloak                string `json:"Keycloak"`
		Mocktitlements          string `json:"mocktitlements"`
		CaddyReverseProxy       string `json:"caddyReverseProxy"`
		ObjectStoreMinio        string `json:"objectStoreMinio"`
		FeatureFlagsUnleash     string `json:"featureFlagsUnleash"`
		FeatureFlagsUnleashEdge string `json:"featureFlagsUnleashEdge"`
		TokenRefresher          string `json:"tokenRefresher"`
		OtelCollector           string `json:"otelCollector"`
		InMemoryDB              string `json:"inMemoryDB"`
		PrometheusGateway       string `json:"prometheusGateway"`
	} `json:"images"`
	DebugOptions struct {
		Logging struct {
			DebugLogging bool `json:"debugLogging"`
		} `json:"logging"`
		Trigger struct {
			Diff bool `json:"diff"`
		} `json:"trigger"`
		Cache struct {
			Create bool `json:"create"`
			Update bool `json:"update"`
			Apply  bool `json:"apply"`
		} `json:"cache"`
		Pprof struct {
			Enable  bool   `json:"enable"`
			CPUFile string `json:"cpuFile"`
		} `json:"pprof"`
	} `json:"debugOptions"`
	Features struct {
		CreateServiceMonitor        bool `json:"createServiceMonitor"`
		DisableWebhooks             bool `json:"disableWebhooks"`
		WatchStrimziResources       bool `json:"watchStrimziResources"`
		UseComplexStrimziTopicNames bool `json:"useComplexStrimziTopicNames"`
		EnableAuthSidecarHook       bool `json:"enableAuthSidecarHook"`
		KedaResources               bool `json:"enableKedaResources"`
		PerProviderMetrics          bool `json:"perProviderMetrics"`
		ReconciliationMetrics       bool `json:"reconciliationMetrics"`
		EnableDependencyMetrics     bool `json:"enableDependencyMetrics"`
		DisableCloudWatchLogging    bool `json:"disableCloudWatchLogging"`
		EnableExternalStrimzi       bool `json:"enableExternalStrimzi"`
		DisableRandomRoutes         bool `json:"disableRandomRoutes"`
		DisableStrimziFinalizer     bool `json:"disableStrimziFinalizer"`
	} `json:"features"`
	Settings struct {
		ManagedKafkaEphemDeleteRegex string `json:"managedKafkaEphemDeleteRegex"`
		RestarterAnnotationName      string `json:"restarterAnnotation"`
	} `json:"settings"`
}

func getConfig() ClowderConfig {
	configPath := "/config/clowder_config.json"

	if path := os.Getenv("CLOWDER_CONFIG_PATH"); path != "" {
		configPath = path
	}

	fmt.Printf("Loading config from: %s\n", configPath)

	jsonData, err := os.ReadFile(configPath)

	if err != nil {
		fmt.Printf("Config file not found\n")
		return ClowderConfig{}
	}

	clowderConfig := ClowderConfig{}
	err = json.Unmarshal(jsonData, &clowderConfig)

	if err != nil {
		fmt.Printf("Couldn't parse json:\n %s", err.Error())
		return ClowderConfig{}
	}

	if clowderConfig.Settings.RestarterAnnotationName == "" {
		clowderConfig.Settings.RestarterAnnotationName = "qontract.recycle"
	}

	return clowderConfig
}

// LoadedConfig is the loaded Clowder config.
var LoadedConfig ClowderConfig

func init() {
	LoadedConfig = getConfig()
}
