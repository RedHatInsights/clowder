package objectstore

import (
	"context"
	errlib "errors"
	"strconv"
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"github.com/stretchr/testify/assert"
)

// TODO: replace with assert.ErrorIs whenever testify is next released...
func assertErrorIs(t *testing.T, got error, want error) {
	t.Helper()
	if !errlib.Is(got, want) {
		t.Errorf("got error: %s, want error: %s", got, want)
	}
}

type mockBucket struct {
	Name        string
	Exists      bool
	CreateError error
	ExistsError error
}

type mockBucketHandler struct {
	hostname              string
	port                  int
	accessKey             *string
	secretKey             *string
	wantCreateClientError bool
	ExistsCalls           []string
	MakeCalls             []string
	MockBuckets           []mockBucket
}

func (c *mockBucketHandler) Exists(ctx context.Context, bucketName string) (bool, error) {
	// track the calls to this mock func
	c.ExistsCalls = append(c.ExistsCalls, bucketName)

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
	// track the calls to this mock func
	c.MakeCalls = append(c.MakeCalls, bucketName)

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
	if c.wantCreateClientError == true {
		return errors.New("create client error")
	}
	c.hostname = hostname
	c.port = port
	c.accessKey = accessKey
	c.secretKey = secretKey
	return nil
}

func getTestProvider(t *testing.T) p.Provider {
	t.Helper()
	return p.Provider{Ctx: context.TODO()}
}

func getTestMinioProvider(t *testing.T) *minioProvider {
	t.Helper()
	testMinioProvider := &minioProvider{
		Provider: getTestProvider(t),
	}
	return testMinioProvider
}

func setupBucketTest(t *testing.T, mockBuckets []mockBucket) (
	*mockBucketHandler, *crd.ClowdApp, *minioProvider,
) {
	t.Helper()
	var bucketNames []string
	for _, mb := range mockBuckets {
		bucketNames = append(bucketNames, mb.Name)
	}
	testApp := &crd.ClowdApp{
		Spec: crd.ClowdAppSpec{
			ObjectStore: bucketNames,
		},
	}
	testMinioProvider := getTestMinioProvider(t)
	testBucketHandler := &mockBucketHandler{MockBuckets: mockBuckets}
	testMinioProvider.BucketHandler = testBucketHandler
	return testBucketHandler, testApp, testMinioProvider
}

func TestMinio(t *testing.T) {
	assert := assert.New(t)

	fakeError := errors.New("something very bad happened")

	t.Run("configure", func(t *testing.T) {
		testAppConfig := &config.AppConfig{}

		mp := getTestMinioProvider(t)
		mp.Config = config.ObjectStoreConfig{
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
		}
		mp.Configure(testAppConfig)
		result := testAppConfig.ObjectStore

		equalsErr := objectStoreEquals(result, &mp.Config)

		if equalsErr != "" {
			t.Error(equalsErr)
		}
	})

	t.Run("createBucketsHitsCheckError", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{mockBucket{
			Name:        bucketName,
			Exists:      false,
			ExistsError: fakeError,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)

		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 0)
		assert.Contains(handler.ExistsCalls, bucketName)
		wantErr := newBucketError(bucketCheckErrorMsg, bucketName, fakeError)
		assert.Error(gotErr)
		assertErrorIs(t, gotErr, wantErr)
	})

	t.Run("createBucketsHitsCreateError", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{mockBucket{
			Name:        bucketName,
			Exists:      false,
			CreateError: fakeError,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)

		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 1)
		assert.Contains(handler.ExistsCalls, bucketName)
		assert.Contains(handler.MakeCalls, bucketName)
		wantErr := newBucketError(bucketCreateErrorMsg, bucketName, fakeError)
		assert.Error(gotErr)
		assertErrorIs(t, gotErr, wantErr)
	})

	t.Run("createBucketsAlreadyExists", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{mockBucket{
			Name:   bucketName,
			Exists: true,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 0)
		assert.Contains(handler.ExistsCalls, bucketName)
	})

	t.Run("createBucketsSuccess", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{mockBucket{
			Name:   bucketName,
			Exists: false,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 1)
		assert.Contains(handler.ExistsCalls, bucketName)
		assert.Contains(handler.MakeCalls, bucketName)
	})

	t.Run("createMultipleBuckets", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			mockBucket{Name: b1, Exists: false},
			mockBucket{Name: b2, Exists: false},
			mockBucket{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 3)
		assert.Len(handler.MakeCalls, 3)
		for _, b := range []string{b1, b2, b3} {
			assert.Contains(handler.ExistsCalls, b)
			assert.Contains(handler.MakeCalls, b)
		}
	})

	t.Run("createMultipleBucketsSomeExist", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			mockBucket{Name: b1, Exists: true},
			mockBucket{Name: b2, Exists: true},
			mockBucket{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 3)
		assert.Len(handler.MakeCalls, 1)
		for _, b := range []string{b1, b2, b3} {
			assert.Contains(handler.ExistsCalls, b)
		}
		assert.Contains(handler.MakeCalls, b3)
	})

	t.Run("createMultipleBucketsWithExistsFailure", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			mockBucket{Name: b1, Exists: false},
			mockBucket{Name: b2, Exists: false, ExistsError: fakeError},
			mockBucket{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)
		wantErr := newBucketError(bucketCheckErrorMsg, b2, fakeError)
		assert.Error(gotErr)
		assertErrorIs(t, wantErr, gotErr)

		// CreateBuckets should have bailed early
		assert.Len(handler.ExistsCalls, 2)
		assert.Len(handler.MakeCalls, 1)
	})

	t.Run("createMultipleBucketsWithCreateFailure", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			mockBucket{Name: b1, Exists: false},
			mockBucket{Name: b2, Exists: false, CreateError: fakeError},
			mockBucket{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)
		gotErr := mp.CreateBuckets(app)
		wantErr := newBucketError(bucketCreateErrorMsg, b2, fakeError)
		assert.Error(gotErr)
		assertErrorIs(t, wantErr, gotErr)

		// CreateBuckets should have bailed early
		assert.Len(handler.ExistsCalls, 2)
		assert.Len(handler.MakeCalls, 2)
	})

	t.Run("minioProviderCreate", func(t *testing.T) {
		secMap := map[string]string{
			"accessKey": "123456abcdef",
			"secretKey": "abcdef123456",
			"hostname":  "foo.bar.svc",
			"port":      "9000",
		}
		tp := getTestProvider(t)

		mp, err := createMinioProvider(
			&tp, secMap, &mockBucketHandler{wantCreateClientError: false},
		)

		assert.NoError(err)
		assert.Equal(mp.Config.Hostname, secMap["hostname"])
		port, _ := strconv.Atoi(secMap["port"])
		assert.Equal(mp.Config.Port, port)
		assert.Equal(mp.Config.AccessKey, p.StrPtr(secMap["accessKey"]))
		assert.Equal(mp.Config.SecretKey, p.StrPtr(secMap["secretKey"]))
		assert.Equal(mp.Ctx, tp.Ctx)
	})

	t.Run("minioProviderCreateHitsError", func(t *testing.T) {
		secMap := map[string]string{
			"accessKey": "123456abcdef",
			"secretKey": "abcdef123456",
			"hostname":  "foo.bar.svc",
			"port":      "9000",
		}
		tp := getTestProvider(t)

		mp, err := createMinioProvider(&tp, secMap, &mockBucketHandler{wantCreateClientError: true})

		assert.Error(err)
		assert.Nil(mp)
	})
}
