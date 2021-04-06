package clowder_config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	configFile, err := os.Open("test_config.json")

	assert.NoError(t, err)

	jsonParser := json.NewDecoder(configFile)

	config := ClowderConfig{}
	err = jsonParser.Decode(&config)

	assert.NoError(t, err)

	debug := config.DebugOptions

	assert.Equal(t, debug.Trigger.Diff, true)
	assert.Equal(t, debug.Cache.Create, true)
	assert.Equal(t, debug.Cache.Update, false)
	assert.Equal(t, debug.Cache.Apply, true)
}
