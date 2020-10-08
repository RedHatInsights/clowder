package database

import (
	"fmt"
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	core "k8s.io/api/core/v1"
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

	configs, err := genDbConfigs(secrets)

	if err != nil {
		t.Error("Failed to gen db config", err)
	}

	if len(configs) != 1 {
		t.Errorf("Wrong number of configs %d; expected 1", len(configs))
		t.FailNow()
	}

	spec := crd.DatabaseSpec{Name: dbName}

	resolved := resolveDb(spec, configs)

	if resolved != configs[0] {
		t.Error("resolveDb did not match given config")
	}
}
