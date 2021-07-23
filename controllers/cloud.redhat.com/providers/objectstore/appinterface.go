package objectstore

import (
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type appInterfaceObjectstoreProvider struct {
	providers.Provider
	Config config.ObjectStoreConfig
}

// NewAppInterfaceObjectstore returns a new app-interface object store provider object.
func NewAppInterfaceObjectstore(p *providers.Provider) (providers.ClowderProvider, error) {
	provider := appInterfaceObjectstoreProvider{Provider: *p}

	return &provider, nil
}

func (a *appInterfaceObjectstoreProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
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

	err = resolveBucketDeps(app.Spec.ObjectStore, objStoreConfig)

	if err != nil {
		return err
	}

	c.ObjectStore = objStoreConfig
	return nil
}

func resolveBucketDeps(requestedBuckets []string, c *config.ObjectStoreConfig) error {
	buckets := []config.ObjectStoreBucket{}
	missing := []string{}

	for _, requestedBucket := range requestedBuckets {
		found := false
		for _, bucket := range c.Buckets {
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

	c.Buckets = buckets
	return nil
}

func genObjStoreConfig(secrets []core.Secret) (*config.ObjectStoreConfig, error) {
	buckets := []config.ObjectStoreBucket{}
	objectStoreConfig := config.ObjectStoreConfig{Port: 443}

	extractFn := func(m map[string][]byte, bucket string) {
		bucketConfig := config.ObjectStoreBucket{
			AccessKey: providers.StrPtr(string(m["aws_access_key_id"])),
			SecretKey: providers.StrPtr(string(m["aws_secret_access_key"])),
			Name:      bucket,
			Region:    providers.StrPtr(string(m["aws_region"])),
		}

		if endpoint, ok := m["endpoint"]; ok {
			objectStoreConfig.Hostname = string(endpoint)
		}

		buckets = append(buckets, bucketConfig)
	}

	extractFnNoAnno := func(m map[string][]byte) {
		extractFn(m, string(m["bucket"]))
	}

	keys := []string{"aws_access_key_id", "aws_secret_access_key"}
	annoKey := "clowder/bucket-names"
	providers.ExtractSecretDataAnno(secrets, extractFn, annoKey, keys...)
	keys = append(keys, "bucket")
	providers.ExtractSecretData(secrets, extractFnNoAnno, keys...)

	if len(buckets) > 0 && objectStoreConfig.Hostname == "" {
		err := errors.New("Could not find object store hostname from secrets")
		return nil, err
	}

	objectStoreConfig.Buckets = buckets
	objectStoreConfig.Tls = true
	return &objectStoreConfig, nil
}
