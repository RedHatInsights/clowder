package inmemorydb

import (
	"fmt"
	"strconv"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type elasticache struct {
	p.Provider
	Config config.InMemoryDBConfig
}

func (e *elasticache) Provide(app *crd.ClowdApp, config *config.AppConfig) error {
	if !app.Spec.InMemoryDB {
		return nil
	}

	secrets := core.SecretList{}
	err := e.Client.List(e.Ctx, &secrets, client.InNamespace(app.Namespace))

	if err != nil {
		msg := fmt.Sprintf("Failed to list secrets in %s", app.Namespace)
		return errors.Wrap(msg, err)
	}

	found := false

	for _, secret := range secrets.Items {
		if secret.Name == "in-memory-db" {
			port, err := strconv.Atoi(string(secret.Data["db.port"]))

			if err != nil {
				return errors.Wrap("Failed to parse im-memory-db port", err)
			}

			e.Config.Hostname = string(secret.Data["db.endpoint"])
			e.Config.Port = port
			found = true
			break
		}
	}

	if found == false {
		return &errors.MissingDependencies{
			MissingDeps: map[string][]string{
				"in-memory-db-secret": {app.Name},
			},
		}
	}

	config.InMemoryDb = &e.Config

	return nil
}

// NewElasticache returns a new elasticache provider object.
func NewElasticache(p *providers.Provider) (providers.ClowderProvider, error) {
	config := config.InMemoryDBConfig{}

	redisProvider := elasticache{Provider: *p, Config: config}

	return &redisProvider, nil
}
