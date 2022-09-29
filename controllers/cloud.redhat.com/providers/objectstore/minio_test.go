package objectstore

import (
	"context"
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func getTestProvider(t *testing.T) providers.Provider {
	t.Helper()
	return providers.Provider{
		Ctx: context.TODO(),
		Env: &crd.ClowdEnvironment{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
		},
		Client: &FakeClient{},
		Config: &config.AppConfig{},
	}
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

type FakeClient struct {
}

type FakeStatus struct {
}

func (fc *FakeClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (fc *FakeClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (fc *FakeClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (fc *FakeClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (fc *FakeClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (fc *FakeClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	p, _ := obj.(*core.Secret)
	p.Data = make(map[string][]byte)
	p.Data["port"] = []byte("2345")
	return nil
}

func (fc *FakeClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (fc *FakeClient) Scheme() *runtime.Scheme {
	return nil
}

func (fc *FakeClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (fc *FakeClient) Status() client.StatusWriter {
	return &FakeStatus{}
}

func (fc *FakeStatus) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (fc *FakeStatus) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func TestMinio(t *testing.T) {
	assert := assert.New(t)

	fakeError := errors.New("something very bad happened")

	t.Run("createBucketsHitsCheckError", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{{
			Name:        bucketName,
			Exists:      false,
			ExistsError: fakeError,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)

		gotErr := mp.Provide(app)
		wantErr := newBucketError(bucketCheckErrorMsg, bucketName, fakeError)
		assert.Error(gotErr)

		assert.Len(mp.Config.ObjectStore.Buckets, 0)
		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 0)
		assert.Contains(handler.ExistsCalls, bucketName)
		assert.ErrorIs(gotErr, wantErr)
	})

	t.Run("createBucketsHitsCreateError", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{{
			Name:        bucketName,
			Exists:      false,
			CreateError: fakeError,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)

		mp.Config = &config.AppConfig{
			ObjectStore: &config.ObjectStoreConfig{
				Hostname: "",
				Port:     0,
				Tls:      false,
			},
		}

		gotErr := mp.Provide(app)
		wantErr := newBucketError(bucketCreateErrorMsg, bucketName, fakeError)
		assert.Error(gotErr)
		assert.ErrorIs(gotErr, wantErr)

		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 1)
		assert.Contains(handler.ExistsCalls, bucketName)
		assert.Contains(handler.MakeCalls, bucketName)
	})

	t.Run("createBucketsAlreadyExists", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{{
			Name:   bucketName,
			Exists: true,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)

		mp.Config = &config.AppConfig{
			ObjectStore: &config.ObjectStoreConfig{
				Hostname: "",
				Port:     0,
				Tls:      false,
			},
		}

		gotErr := mp.Provide(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 0)
		assert.Contains(handler.ExistsCalls, bucketName)

		wantBucketConfig := config.ObjectStoreBucket{Name: bucketName, RequestedName: bucketName}
		assert.Contains(mp.Config.ObjectStore.Buckets, wantBucketConfig)
		assert.Len(mp.Config.ObjectStore.Buckets, 1)
	})

	t.Run("createBucketsSuccess", func(t *testing.T) {
		bucketName := "testBucket"
		mockBuckets := []mockBucket{{
			Name:   bucketName,
			Exists: false,
		}}
		handler, app, mp := setupBucketTest(t, mockBuckets)

		mp.Config = &config.AppConfig{
			ObjectStore: &config.ObjectStoreConfig{
				Hostname: "",
				Port:     0,
				Tls:      false,
			},
		}

		gotErr := mp.Provide(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 1)
		assert.Len(handler.MakeCalls, 1)
		assert.Contains(handler.ExistsCalls, bucketName)
		assert.Contains(handler.MakeCalls, bucketName)

		wantBucketConfig := config.ObjectStoreBucket{Name: bucketName, RequestedName: bucketName}
		assert.Contains(mp.Config.ObjectStore.Buckets, wantBucketConfig)
		assert.Len(mp.Config.ObjectStore.Buckets, 1)
	})

	t.Run("createMultipleBuckets", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			{Name: b1, Exists: false},
			{Name: b2, Exists: false},
			{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)

		mp.Config = &config.AppConfig{
			ObjectStore: &config.ObjectStoreConfig{
				Hostname: "",
				Port:     0,
				Tls:      false,
			},
		}

		gotErr := mp.Provide(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 3)
		assert.Len(handler.MakeCalls, 3)
		assert.Len(mp.Config.ObjectStore.Buckets, 3)
		for _, b := range []string{b1, b2, b3} {
			wantBucketConfig := config.ObjectStoreBucket{Name: b, RequestedName: b}
			assert.Contains(mp.Config.ObjectStore.Buckets, wantBucketConfig)
			assert.Contains(handler.ExistsCalls, b)
			assert.Contains(handler.MakeCalls, b)
		}
	})

	t.Run("createMultipleBucketsSomeExist", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			{Name: b1, Exists: true},
			{Name: b2, Exists: true},
			{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)
		mp.Config = &config.AppConfig{
			ObjectStore: &config.ObjectStoreConfig{
				Hostname: "",
				Port:     0,
				Tls:      false,
			},
		}

		gotErr := mp.Provide(app)
		assert.NoError(gotErr)
		assert.Len(handler.ExistsCalls, 3)
		assert.Len(handler.MakeCalls, 1)
		assert.Len(mp.Config.ObjectStore.Buckets, 3)
		for _, b := range []string{b1, b2, b3} {
			assert.Contains(handler.ExistsCalls, b)
			wantBucketConfig := config.ObjectStoreBucket{Name: b, RequestedName: b}
			assert.Contains(mp.Config.ObjectStore.Buckets, wantBucketConfig)
		}
		assert.Contains(handler.MakeCalls, b3)
	})

	t.Run("createMultipleBucketsWithExistsFailure", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			{Name: b1, Exists: false},
			{Name: b2, Exists: false, ExistsError: fakeError},
			{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)
		mp.Config = &config.AppConfig{
			ObjectStore: &config.ObjectStoreConfig{
				Hostname: "",
				Port:     0,
				Tls:      false,
			},
		}

		gotErr := mp.Provide(app)
		wantErr := newBucketError(bucketCheckErrorMsg, b2, fakeError)
		assert.Error(gotErr)
		assert.ErrorIs(gotErr, wantErr)

		// Provide should have bailed early
		assert.Len(handler.ExistsCalls, 2)
		assert.Len(handler.MakeCalls, 1)
		assert.Len(mp.Config.ObjectStore.Buckets, 1)
		wantBucketConfig := config.ObjectStoreBucket{Name: b1, RequestedName: b1}
		assert.Contains(mp.Config.ObjectStore.Buckets, wantBucketConfig)
	})

	t.Run("createMultipleBucketsWithCreateFailure", func(t *testing.T) {
		b1, b2, b3 := "testBucket1", "testBucket2", "testBucket3"

		mockBuckets := []mockBucket{
			{Name: b1, Exists: false},
			{Name: b2, Exists: false, CreateError: fakeError},
			{Name: b3, Exists: false},
		}

		handler, app, mp := setupBucketTest(t, mockBuckets)
		mp.Config = &config.AppConfig{
			ObjectStore: &config.ObjectStoreConfig{
				Hostname: "",
				Port:     0,
				Tls:      false,
			},
		}

		gotErr := mp.Provide(app)
		wantErr := newBucketError(bucketCreateErrorMsg, b2, fakeError)
		assert.Error(gotErr)
		assert.ErrorIs(gotErr, wantErr)

		// Provide should have bailed early
		assert.Len(handler.ExistsCalls, 2)
		assert.Len(handler.MakeCalls, 2)
		assert.Len(mp.Config.ObjectStore.Buckets, 1)
		wantBucketConfig := config.ObjectStoreBucket{Name: b1, RequestedName: b1}
		assert.Contains(mp.Config.ObjectStore.Buckets, wantBucketConfig)
	})
}
