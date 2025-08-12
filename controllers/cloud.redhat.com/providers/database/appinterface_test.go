package database

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
)

func TestAppInterfaceDb(t *testing.T) {
	dbName := "test-db"
	secrets := []core.Secret{{
		Data: map[string][]byte{
			"db.host":     []byte(fmt.Sprintf("%s-prod.amazing.aws.amazon.com", dbName)),
			"db.port":     []byte("5432"),
			"db.user":     []byte("user"),
			"db.password": []byte("password"),
			"db.name":     []byte(dbName),
		},
	}}

	configs, err := genDbConfigs(secrets, false)

	assert.NoError(t, err, "failed to gen db config")
	assert.Equal(t, len(configs), 1, "wrong number of configs")

	spec := crd.DatabaseSpec{Name: dbName}

	resolved := resolveDb(spec, configs)

	assert.Equal(t, configs[0], resolved, "resolveDb did not match given config")
}
