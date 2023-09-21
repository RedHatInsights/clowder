package objectstore

import (
	"testing"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type TestSecrets struct {
	ExactKeys   map[string]string
	ExtraKeys   map[string]string
	NoKeys      map[string]string
	MissingKeys map[string]string
}

func (ts *TestSecrets) NewValidConfig(name string) map[string]string {
	return map[string]string{
		"aws_access_key_id":     ts.ExactKeys["aws_access_key_id"],
		"aws_secret_access_key": ts.ExactKeys["aws_secret_access_key"],
		"aws_region":            ts.ExactKeys["aws_region"],
		"bucket":                name,
	}
}

func (ts *TestSecrets) ToSecrets() []core.Secret {
	theMaps := []map[string]string{
		ts.ExactKeys,
		ts.ExtraKeys,
		ts.NoKeys,
		ts.MissingKeys,
	}

	secrets := []core.Secret{}

	for _, secMap := range theMaps {
		bytemap := map[string][]byte{}

		for k, v := range secMap {
			bytemap[k] = []byte(v)
		}

		secrets = append(secrets, core.Secret{
			Data: bytemap,
		})
	}

	return secrets
}

func TestAppInterfaceObjectStore(t *testing.T) {
	testSecretSpecs := TestSecrets{
		ExactKeys: map[string]string{
			"aws_access_key_id":     utils.RandString(12),
			"aws_secret_access_key": utils.RandString(12),
			"aws_region":            "us-east-1",
			"bucket":                "test-bucket",
			"endpoint":              "s3.us-east-1.aws.amazon.com",
		},
	}

	c, err := genObjStoreConfig(testSecretSpecs.ToSecrets())

	assert.NoError(t, err, "error calling genObjStoreConfig")

	expected := config.ObjectStoreConfig{
		Port:     443,
		Hostname: testSecretSpecs.ExactKeys["endpoint"],
		Buckets: []config.ObjectStoreBucket{{
			Region:    utils.StringPtr("us-east-1"),
			AccessKey: utils.StringPtr(testSecretSpecs.ExactKeys["aws_access_key_id"]),
			SecretKey: utils.StringPtr(testSecretSpecs.ExactKeys["aws_secret_access_key"]),
			Name:      testSecretSpecs.ExactKeys["bucket"],
			Endpoint:  utils.StringPtr("s3.us-east-1.aws.amazon.com"),
			Tls:       utils.TruePtr(),
		}},
		Tls: true,
	}

	assert.Equal(t, &expected, c)
}
