package objectstore

import (
	"context"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MinIO struct {
	Client *minio.Client
}

func GetConfig(ctx context.Context, m crd.MinioStatus, c client.Client) (*config.ObjectStoreConfig, error) {
	conf := &config.ObjectStoreConfig{
		Endpoint: m.Endpoint,
	}
	name := types.NamespacedName{
		Name:      m.Credentials.Name,
		Namespace: m.Credentials.Namespace,
	}
	secret := core.Secret{}
	if err := c.Get(ctx, name, &secret); err != nil {
		return conf, err
	}

	conf.AccessKey = string(secret.Data["accessKey"])
	conf.SecretKey = string(secret.Data["secretKey"])

	return conf, nil
}

// NewMinIO constructs a new client for the given config
func NewMinIO(cfg *config.ObjectStoreConfig) (*MinIO, error) {
	cl, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	return &MinIO{
		Client: cl,
	}, nil
}

// CreateBucket creates a new bucket
func (m *MinIO) CreateBucket(ctx context.Context, bucket string) error {
	found, err := m.Client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if found {
		return nil // possibly return a found error?
	}
	return m.Client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}
