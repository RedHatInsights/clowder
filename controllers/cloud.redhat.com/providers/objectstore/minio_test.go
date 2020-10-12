package objectstore

import (
	"testing"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

func TestMinioConfigure(t *testing.T) {
	testProvider := &minioProvider{
		Config: config.ObjectStoreConfig{
			Buckets: []config.ObjectStoreBucket{{
				AccessKey:     p.StrPtr("bucket_access_key"),
				Name:          "my_bucket",
				RequestedName: "my_bucket_requested_name",
				SecretKey:     p.StrPtr("bucket_secret_key"),
			}},
			Hostname:  "my.minio.com",
			Port:      8080,
			AccessKey: p.StrPtr("access_key"),
			SecretKey: p.StrPtr("secret_key"),
		},
	}

	testAppConfig := &config.AppConfig{}

	testProvider.Configure(testAppConfig)
	result := testAppConfig.ObjectStore

	equalsErr := objectStoreEquals(result, &testProvider.Config)

	if equalsErr != "" {
		t.Error(equalsErr)
	}
}
