package objectstore

import (
	"fmt"
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppInterfaceObjectstoreProvider struct {
	p.Provider
	Config config.ObjectStoreConfig
}

func (a *AppInterfaceObjectstoreProvider) Configure(c *config.AppConfig) {
	c.ObjectStore = &a.Config
}

func NewAppInterfaceObjectstore(p *p.Provider) (ObjectStoreProvider, error) {
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

	resolveBucketDeps(app.Spec.ObjectStore, objStoreConfig)
	a.Config = *objStoreConfig
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

	for _, secret := range secrets {
		accessKey, accessKeyOk := secret.Data["aws_access_key_id"]
		secretKey, secretKeyOk := secret.Data["aws_secret_access_key"]
		name, nameOk := secret.Data["bucket"]
		endpoint, endpointOk := secret.Data["endpoint"]

		if accessKeyOk && secretKeyOk && nameOk {
			bucketConfig := config.ObjectStoreBucket{
				AccessKey: p.StrPtr(string(accessKey)),
				SecretKey: p.StrPtr(string(secretKey)),
				Name:      string(name),
			}
			if endpointOk {
				objectStoreConfig.Hostname = string(endpoint)
			}
			buckets = append(buckets, bucketConfig)
		}
	}

	if len(buckets) > 0 && objectStoreConfig.Hostname == "" {
		err := errors.New("Could not find object store hostname from secrets")
		return nil, err
	}

	objectStoreConfig.Buckets = buckets
	return &objectStoreConfig, nil
}
