package providers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectStoreProvider interface {
	CreateBucket(ctx context.Context, bucket string) error
}

func Get(ctx context.Context, client *client.Client, kind string, env crd.ClowdEnvironment) (ObjectStoreProvider, ctrl.Result, error) {

	var result ctrl.Result
	var provider Provider

	providerCtx := ProviderContext{
		Client: *client,
		Ctx:    ctx,
		Env:    &env,
	}

	switch kind {
	case "minio":
		provider = MinIO{}
		result, err := provider.Init(providerCtx)

		if err != nil {
			return provider, result, err
		}
	}

	return provider, result, nil
}
