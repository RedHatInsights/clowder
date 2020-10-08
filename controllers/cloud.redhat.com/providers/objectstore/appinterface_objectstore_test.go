package objectstore

import (
	"fmt"
	"testing"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
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

	if err != nil {
		t.Errorf("Error calling genObjStoreConfig: %w", err)
	}

	expected := config.ObjectStoreConfig{
		Port:     443,
		Hostname: testSecretSpecs.ExactKeys["endpoint"],
		Buckets: []config.ObjectStoreBucket{{
			AccessKey: p.StrPtr(testSecretSpecs.ExactKeys["aws_access_key_id"]),
			SecretKey: p.StrPtr(testSecretSpecs.ExactKeys["aws_secret_access_key"]),
			Name:      testSecretSpecs.ExactKeys["bucket"],
		}},
	}

	equalsErr := objectStoreEquals(c, &expected)

	if equalsErr != "" {
		t.Error(equalsErr)
	}
}

func objectStoreEquals(actual *config.ObjectStoreConfig, expected *config.ObjectStoreConfig) string {
	oneNil, otherNil := actual == nil, expected == nil

	if oneNil && otherNil {
		return ""
	}

	if oneNil != otherNil {
		return "One object is nil"
	}

	actualLen, expectedLen := len(actual.Buckets), len(expected.Buckets)

	if actualLen != expectedLen {
		return fmt.Sprintf("Different number of buckets %d; expected %d", actualLen, expectedLen)
	}

	for i, bucket := range actual.Buckets {
		expectedBucket := expected.Buckets[i]
		if bucket.Name != expectedBucket.Name {
			return fmt.Sprintf("Bad bucket name %s; expected %s", bucket.Name, expectedBucket.Name)
		}
		if *bucket.AccessKey != *expectedBucket.AccessKey {
			return fmt.Sprintf(
				"%s: Bad accessKey '%s'; expected '%s'",
				bucket.Name,
				*bucket.AccessKey,
				*expectedBucket.AccessKey,
			)
		}
		if *bucket.SecretKey != *expectedBucket.SecretKey {
			return fmt.Sprintf(
				"%s: Bad secretKey %s; expected %s",
				bucket.Name,
				*bucket.SecretKey,
				*expectedBucket.SecretKey,
			)
		}
		if bucket.RequestedName != expectedBucket.RequestedName {
			return fmt.Sprintf(
				"%s: Bad requestedName %s; expected %s",
				bucket.Name,
				bucket.RequestedName,
				expectedBucket.RequestedName,
			)
		}
	}

	if actual.Port != expected.Port {
		return fmt.Sprintf("Bad port %d; expected %d", actual.Port, expected.Port)
	}

	if actual.Hostname != expected.Hostname {
		return fmt.Sprintf("Bad hostname %s; expected %s", actual.Hostname, expected.Hostname)
	}

	if actual.AccessKey != expected.AccessKey {
		return fmt.Sprintf("Bad accessKey %s; expected %s", *actual.AccessKey, *expected.AccessKey)
	}

	if actual.SecretKey != expected.SecretKey {
		return fmt.Sprintf("Bad secretKey %s; expected %s", *actual.SecretKey, *expected.SecretKey)
	}

	return ""
}
