package providers

import (
	"fmt"
	"reflect"
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppInterfaceObjectstoreProvider struct {
	Provider
	Config config.ObjectStoreConfig
}

func (a *AppInterfaceObjectstoreProvider) Configure(c *config.AppConfig) {
	c.ObjectStore = &a.Config
}

func NewAppInterfaceObjectstore(p *Provider) (ObjectStoreProvider, error) {
	provider := AppInterfaceObjectstoreProvider{Provider: *p}

	return &provider, nil
}

func (a *AppInterfaceObjectstoreProvider) CreateBuckets(app *crd.ClowdApp) error {
	if len(app.Spec.ObjectStore) == 0 {
		return nil
	}

	secrets := core.SecretList{}
	err := a.Client.List(a.Ctx, &secrets, client.InNamespace(app.Namespace))

	if err != nil {
		msg := fmt.Sprintf("Failed to list secrets in %s", app.Namespace)
		return errors.Wrap(msg, err)
	}

	objStoreConfig, err := genObjStoreConfig(secrets.Items)

	if err != nil {
		return err
	}

	buckets := []config.ObjectStoreBucket{}
	missing := []string{}

	for _, requestedBucket := range app.Spec.ObjectStore {
		found := false
		for _, bucket := range objStoreConfig.Buckets {
			if strings.HasPrefix(bucket.Name, requestedBucket) {
				found = true
				bucket.RequestedName = requestedBucket
				buckets = append(buckets, bucket)
				break
			}
		}

		if !found {
			missing = append(missing, requestedBucket)
		}
	}

	if len(missing) > 0 {
		bucketStr := strings.Join(missing, ", ")
		return errors.New("Missing buckets from app-interface: " + bucketStr)
	}

	objStoreConfig.Buckets = buckets
	a.Config = *objStoreConfig

	return nil
}

func genObjStoreConfig(secrets []core.Secret) (*config.ObjectStoreConfig, error) {
	buckets := []config.ObjectStoreBucket{}

	objectStoreConfig := config.ObjectStoreConfig{
		Buckets: buckets,
		Port:    443,
	}

	keys := map[string]string{
		"aws_access_key_id":     "AccessKey",
		"aws_secret_access_key": "SecretKey",
		"bucket":                "Name",
	}

	for _, secret := range secrets {
		bucketConfig := config.ObjectStoreBucket{}
		rBucketConfig := reflect.ValueOf(&bucketConfig).Elem()
		found := true
		for key, kVal := range keys {
			if val, ok := secret.Data[key]; ok {
				fv := rBucketConfig.FieldByName(kVal)
				fv.SetString(string(val))
			} else {
				found = false
				break
			}
		}

		if found {
			if val, ok := secret.Data["endpoint"]; ok {
				objectStoreConfig.Hostname = string(val)
			}
			buckets = append(buckets, bucketConfig)
		}
	}

	if len(buckets) > 0 && objectStoreConfig.Hostname == "" {
		err := errors.New("Could not find object store hostname from secrets")
		return &objectStoreConfig, err
	}

	return &objectStoreConfig, nil
}
