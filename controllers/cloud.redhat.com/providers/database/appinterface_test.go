package database

import (
	"fmt"
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
)

func TestAppInterfaceDb(t *testing.T) {
	dbName := "test-db"
	dbNameWithCa := "test-db-with-ca"
	secrets := []core.Secret{{
		Data: map[string][]byte{
			"db.host":     []byte(fmt.Sprintf("%s-prod.amazing.aws.amazon.com", dbName)),
			"db.port":     []byte("5432"),
			"db.user":     []byte("user"),
			"db.password": []byte("password"),
			"db.name":     []byte(dbName),
		},
	},
		{
			Data: map[string][]byte{
				"db.host":     []byte(fmt.Sprintf("%s-prod.amazing.aws.amazon.com", dbNameWithCa)),
				"db.port":     []byte("5432"),
				"db.user":     []byte("user"),
				"db.password": []byte("password"),
				"db.name":     []byte(dbNameWithCa),
				"db.ca_cert":  []byte("im-a-cert"),
			},
		},
	}

	configs, err := genDbConfigs(secrets, false)

	assert.NoError(t, err, "failed to gen db config")
	assert.Equal(t, len(configs), 2, "wrong number of configs")

	spec := crd.DatabaseSpec{Name: dbName}

	resolved := resolveDb(spec, configs)

	assert.Equal(t, configs[0], resolved, "resolveDb did not match given config")
	assert.Nil(t, resolved.Config.RdsCa, nil)

	specWithCa := crd.DatabaseSpec{Name: dbNameWithCa}
	resolvedWithCa := resolveDb(specWithCa, configs)

	assert.Equal(t, configs[1], resolvedWithCa, "resolveDb did not match given config")
	assert.NotNil(t, resolvedWithCa.Config.RdsCa)
	assert.Equal(t, *resolvedWithCa.Config.RdsCa, "im-a-cert")

}
