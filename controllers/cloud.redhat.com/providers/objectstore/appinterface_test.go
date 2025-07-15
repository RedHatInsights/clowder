package objectstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"

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

	err = resolveBucketDeps([]string{"test-bucket"}, c)

	assert.NoError(t, err, "error calling resolveBucketDeps")

	expected := config.ObjectStoreConfig{
		Port:     443,
		Hostname: testSecretSpecs.ExactKeys["endpoint"],
		Buckets: []config.ObjectStoreBucket{{
			Region:        utils.StringPtr("us-east-1"),
			AccessKey:     utils.StringPtr(testSecretSpecs.ExactKeys["aws_access_key_id"]),
			SecretKey:     utils.StringPtr(testSecretSpecs.ExactKeys["aws_secret_access_key"]),
			Name:          testSecretSpecs.ExactKeys["bucket"],
			Endpoint:      utils.StringPtr(testSecretSpecs.ExactKeys["endpoint"]),
			Tls:           utils.TruePtr(),
			RequestedName: testSecretSpecs.ExactKeys["bucket"],
		}},
		Tls: true,
	}

	assert.Equal(t, &expected, c)
}

func TestAppInterfaceBadBucket(t *testing.T) {
	c := config.ObjectStoreConfig{
		Hostname: "original",
		Buckets: []config.ObjectStoreBucket{{
			AccessKey:     utils.StringPtr("access"),
			Endpoint:      utils.StringPtr("endpoint"),
			Name:          "badger",
			Region:        utils.StringPtr("region"),
			RequestedName: "badger",
			SecretKey:     utils.StringPtr("secret"),
			Tls:           utils.TruePtr(),
		}},
	}
	err := resolveBucketDeps([]string{"test-bucket"}, &c)

	assert.Error(t, err)
}

func TestAppInterfaceGoodAndBadBucket(t *testing.T) {
	c := config.ObjectStoreConfig{
		Hostname: "original",
		Buckets: []config.ObjectStoreBucket{
			{
				AccessKey:     utils.StringPtr("access"),
				Endpoint:      utils.StringPtr("endpoint"),
				Name:          "badger",
				Region:        utils.StringPtr("region"),
				RequestedName: "badger",
				SecretKey:     utils.StringPtr("secret"),
				Tls:           utils.TruePtr(),
			}, {
				AccessKey:     utils.StringPtr("access"),
				Endpoint:      utils.StringPtr("test-endpoint"),
				Name:          "test-bucket",
				Region:        utils.StringPtr("region"),
				RequestedName: "test-bucket",
				SecretKey:     utils.StringPtr("secret"),
				Tls:           utils.TruePtr(),
			},
			{
				AccessKey:     utils.StringPtr("access"),
				Endpoint:      utils.StringPtr("endpoint"),
				Name:          "badgerRage",
				Region:        utils.StringPtr("region"),
				RequestedName: "badgerRage",
				SecretKey:     utils.StringPtr("secret"),
				Tls:           utils.TruePtr(),
			},
		},
	}
	err := resolveBucketDeps([]string{"test-bucket"}, &c)

	assert.NoError(t, err)
	assert.Len(t, c.Buckets, 1)
	assert.Equal(t, c.Hostname, "test-endpoint")
}
