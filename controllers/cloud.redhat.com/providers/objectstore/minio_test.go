package objectstore

import (
	"context"
	errlib "errors"
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type mockBucket struct {
	Name        string
	Exists      bool
	CreateError error
	ExistsError error
}

type mockBucketHandler struct {
	MockBuckets []mockBucket
}

func (c *mockBucketHandler) Exists(ctx context.Context, bucketName string) (bool, error) {
	for _, mockBucket := range c.MockBuckets {
		if mockBucket.Name == bucketName {
			if mockBucket.ExistsError == nil {
				return mockBucket.Exists, nil
			}
			return mockBucket.Exists, mockBucket.ExistsError
		}
	}
	// todo: really we should error out of the test here if there's no MockBuckets
	return false, nil
}

func (c *mockBucketHandler) Make(ctx context.Context, bucketName string) (err error) {
	for _, mockBucket := range c.MockBuckets {
		if mockBucket.Name == bucketName {
			if mockBucket.CreateError == nil {
				return nil
			}
			return mockBucket.CreateError
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

	bucketWithExistsError := "bucket_with_exists_error"
	fakeError := errors.New("something very bad happened")

	testBucketHandler := &mockBucketHandler{
		MockBuckets: []mockBucket{
			mockBucket{
				Name:        bucketWithExistsError,
				Exists:      false,
				ExistsError: fakeError,
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
			ObjectStore: []string{bucketWithExistsError},
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
		gotErr := testMinioProvider.CreateBuckets(testApp)
		wantErr := newBucketError(bucketCheckErrorMsg, bucketWithExistsError, fakeError)
		if gotErr == nil {
			t.Errorf("Expected to hit an error checking if bucket exists, got nil")
		}
		if !errlib.Is(gotErr, wantErr) {
			t.Errorf("Expected to hit bucket check error, got: %s", gotErr)
		}
	})
}
