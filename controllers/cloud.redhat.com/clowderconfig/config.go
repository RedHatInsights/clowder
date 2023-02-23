package clowderconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

type ClowderConfig struct {
	Images struct {
		MBOP           string `json:"mbop"`
		Caddy          string `json:"caddy"`
		Keycloak       string `json:"Keycloak"`
		Mocktitlements string `json:"mocktitlements"`
		Envoy          string `json:"envoy"`
	} `json:"images"`
	DebugOptions struct {
		Logging struct {
			DebugLogging bool `json:"debugLogging"`
		}
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
		DisableCloudWatchLogging    bool `json:"disableCloudWatchLogging"`
		EnableExternalStrimzi       bool `json:"enableExternalStrimzi"`
		DisableRandomRoutes         bool `json:"disableRandomRoutes"`
	} `json:"features"`
	Settings struct {
		ManagedKafkaEphemDeleteRegex string `json:"managedKafkaEphemDeleteRegex"`
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
		fmt.Printf("Couldn't parse json:\n" + err.Error())
		return ClowderConfig{}
	}

	return clowderConfig
}

var LoadedConfig ClowderConfig

func init() {
	LoadedConfig = getConfig()
}
