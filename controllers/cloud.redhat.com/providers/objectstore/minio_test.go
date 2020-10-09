package objectstore

import (
	"testing"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

func TestMinioConfigure(t *testing.T) {
	testProvider := &minioProvider{
		Buckets: []config.ObjectStoreBucket{{
			AccessKey:     p.StrPtr("bucket_access_key"),
			Name:          "my_bucket",
			RequestedName: "my_bucket_requested_name",
			SecretKey:     p.StrPtr("bucket_secret_key"),
		}},
		Hostname:  "my.minio.com",
		Port:      8080,
		AccessKey: "access_key",
		SecretKey: "secret_key",
	}

	testAppConfig := &config.AppConfig{}

	expected := &config.ObjectStoreConfig{
		Buckets:   testProvider.Buckets,
		Hostname:  testProvider.Hostname,
		Port:      testProvider.Port,
		AccessKey: &testProvider.AccessKey,
		SecretKey: &testProvider.SecretKey,
	}

	testProvider.Configure(testAppConfig)
	result := testAppConfig.ObjectStore

	equalsErr := objectStoreEquals(result, expected)

	if equalsErr != "" {
		t.Error(equalsErr)
	}
}
