package objectstore

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type appInterfaceObjectstoreProvider struct {
	providers.Provider
}

// NewAppInterfaceObjectstore returns a new app-interface object store provider object.
func NewAppInterfaceObjectstore(p *providers.Provider) (providers.ClowderProvider, error) {
	return &appInterfaceObjectstoreProvider{Provider: *p}, nil
}

func (a *appInterfaceObjectstoreProvider) EnvProvide() error {
	return nil
}

func (a *appInterfaceObjectstoreProvider) Provide(app *crd.ClowdApp) error {
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

	a.Config.ObjectStore = objStoreConfig
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
				if *bucket.Endpoint != "" {
					c.Hostname = *bucket.Endpoint
				}
				break
			}
		}

		if !found {
			missing = append(missing, requestedBucket)
		}
	}

	if len(buckets) > 0 && c.Hostname == "" {
		err := errors.NewClowderError("Could not find object store hostname from secrets")
		return err
	}

	if len(missing) > 0 {
		bucketStr := strings.Join(missing, ", ")
		return errors.NewClowderError("Missing buckets from app-interface: " + bucketStr)
	}

	c.Buckets = buckets
	return nil
}

func genObjStoreConfig(secrets []core.Secret) (*config.ObjectStoreConfig, error) {
	buckets := []config.ObjectStoreBucket{}
	objectStoreConfig := config.ObjectStoreConfig{Port: 443}

	extractFn := func(secret *core.Secret, bucket string) {
		bucketConfig := config.ObjectStoreBucket{
			AccessKey: utils.StringPtr(string(secret.Data["aws_access_key_id"])),
			SecretKey: utils.StringPtr(string(secret.Data["aws_secret_access_key"])),
			Name:      bucket,
			Region:    utils.StringPtr(string(secret.Data["aws_region"])),
			Endpoint:  utils.StringPtr(string(secret.Data["endpoint"])),
			Tls:       utils.TruePtr(),
		}

		buckets = append(buckets, bucketConfig)
	}

	extractFnNoAnno := func(secret *core.Secret) {
		extractFn(secret, string(secret.Data["bucket"]))
	}

	keys := []string{"aws_access_key_id", "aws_secret_access_key"}
	annoKey := "clowder/bucket-names"
	providers.ExtractSecretDataAnno(secrets, extractFn, annoKey, keys...)
	keys = append(keys, "bucket")
	providers.ExtractSecretData(secrets, extractFnNoAnno, keys...)

	objectStoreConfig.Buckets = buckets
	objectStoreConfig.Tls = true
	return &objectStoreConfig, nil
}
