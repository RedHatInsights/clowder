package providers

import (
	"context"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProviderContext struct {
	Client client.Client
	Ctx    context.Context
	Env    *crd.ClowdEnvironment
}

type Provider interface {
	Configure(c *config.AppConfig)
	Init(ctx ProviderContext) error
}
