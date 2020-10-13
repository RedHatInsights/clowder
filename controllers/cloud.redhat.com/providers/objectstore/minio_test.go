package objectstore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type mockBucket struct {
	Name           string
	Exists         bool
	CreateErrorMsg string
	ExistsErrorMsg string
}

type mockBucketHandler struct {
	MockBuckets []mockBucket
}

func (c *mockBucketHandler) Exists(ctx context.Context, bucketName string) (bool, error) {
	for _, mockBucket := range c.MockBuckets {
		if mockBucket.Name == bucketName {
			if mockBucket.ExistsErrorMsg == "" {
				return mockBucket.Exists, nil
			}
			return mockBucket.Exists, fmt.Errorf(mockBucket.ExistsErrorMsg)
		}
	}
	// todo: really we should error out of the test here if there's no MockBuckets
	return false, nil
}

func (c *mockBucketHandler) Make(ctx context.Context, bucketName string) (err error) {
	for _, mockBucket := range c.MockBuckets {
		if mockBucket.Name == bucketName {
			if mockBucket.CreateErrorMsg == "" {
				return nil
			}
			return fmt.Errorf(mockBucket.CreateErrorMsg)
		}
	}
	// todo: really we should error out of the test here if there's no MockBuckets
	return nil
}

func (c *mockBucketHandler) CreateClient(
	hostname string, port int, accessKey *string, secretKey *string,
) error {
	return nil
}

func TestMinio(t *testing.T) {
	testProvider := p.Provider{
		Ctx: context.TODO(),
	}

	testBucketHandler := &mockBucketHandler{
		MockBuckets: []mockBucket{
			mockBucket{
				Name:           "bucket_with_exists_error",
				Exists:         false,
				ExistsErrorMsg: "AHH!",
				CreateErrorMsg: "",
			},
		},
	}

	testMinioProvider := &minioProvider{
		Provider:      testProvider,
		BucketHandler: testBucketHandler,
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

	testApp := &crd.ClowdApp{
		Spec: crd.ClowdAppSpec{
			ObjectStore: []string{"bucket_with_exists_error"},
		},
	}

	t.Run("configure", func(t *testing.T) {
		testMinioProvider.Configure(testAppConfig)
		result := testAppConfig.ObjectStore

		equalsErr := objectStoreEquals(result, &testMinioProvider.Config)

		if equalsErr != "" {
			t.Error(equalsErr)
		}
	})

	t.Run("createBucketsHitsCheckError", func(t *testing.T) {
		err := testMinioProvider.CreateBuckets(testApp)
		if err == nil {
			t.Errorf("Expected to hit an error checking if bucket exists, got nil error")
		}
		if !errors.Is(err, fmt.Errorf(fmt.Sprintf(bucketCheckErrorMsg, "bucket_with_exists_error"))) {
			t.Errorf("Expected to hit error of type 'bucketCheckErrorMsg', got: %s", err)
		}
	})

}
