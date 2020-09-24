package providers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Get(ctx context.Context, client *client.Client, kind string, env crd.ClowdEnvironment) (ObjectStoreProvider, error) {

	var provider Provider

	providerCtx := ProviderContext{
		Client: *client,
		Ctx:    ctx,
		Env:    &env,
	}

	switch kind {
	case "minio":
		provider = &MinIO{}
		err := provider.New(providerCtx)

		if err != nil {
			return ObjectStoreProvider(provider), err
		}
	}

	return ObjectStoreProvider(provider), nil
}
