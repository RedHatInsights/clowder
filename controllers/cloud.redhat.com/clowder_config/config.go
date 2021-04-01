package clowder_config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ClowderConfig struct {
	DebugOptions struct {
		Trigger struct {
			Diff bool `json:"diff"`
		} `json:"trigger"`
		Cache struct {
			Create bool `json:"create"`
			Update bool `json:"update"`
			Apply  bool `json:"apply"`
		} `json:"cache"`
	} `json:"debugOptions"`
}

func getConfig() ClowderConfig {
	jsonData, err := ioutil.ReadFile("/config/test")

	if err != nil {
		fmt.Printf("Config file not found")
		return ClowderConfig{}
	}

	clowderConfig := ClowderConfig{}
	err = json.Unmarshal(jsonData, &clowderConfig)

	if err != nil {
		fmt.Printf("Couldn't parse json")
		return ClowderConfig{}
	}

	return clowderConfig
}

var LoadedConfig ClowderConfig

func init() {
	LoadedConfig = getConfig()
}
