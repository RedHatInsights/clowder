package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbasePopulate(t *testing.T) {
	inputData := map[string]string{
		"port":     "9000",
		"hostname": "hostname",
		"name":     "name",
		"password": "password",
		"pgPass":   "pgPass",
		"username": "username",
	}

	config := &DatabaseConfig{}
	config.Populate(&inputData)
	assert.Equal(t, inputData["port"], "9000", "they should be equal")
	assert.Equal(t, inputData["hostname"], "hostname", "they should be equal")
	assert.Equal(t, inputData["username"], "username", "they should be equal")
	assert.Equal(t, inputData["password"], "password", "they should be equal")
	assert.Equal(t, inputData["pgPass"], "pgPass", "they should be equal")
	assert.Equal(t, inputData["name"], "name", "they should be equal")
}
