package inmemorydb

import (
	"fmt"
	"strconv"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type elasticache struct {
	providers.Provider
	Config config.InMemoryDBConfig
}

func (e *elasticache) Provide(app *crd.ClowdApp, config *config.AppConfig) error {
	secretName := "in-memory-db"

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
		if secret.Name == secretName {
			port, err := strconv.Atoi(string(secret.Data["db.port"]))

			if err != nil {
				return errors.Wrap(
					fmt.Sprintf("failed to parse port from secret '%s' in namespace '%s'", secretName, app.Namespace),
					err,
				)
			}

			passwd, err := strconv.Atoi(string(secret.Data["db.auth_token"]))
                        if err != nil {
				// Elasticache PW not found in secret
                                return errors.Wrap(
                                        fmt.Sprintf("Auth token was not found in secret '%s' in namespace '%s'", secretName, app.Namespace),
                                        err,
                                )
                        } else {
				// Elasticache PW found
				e.Config.Password = passwd
			}

			e.Config.Hostname = string(secret.Data["db.endpoint"])
			e.Config.Port = port
			found = true
			break
		}
	}

	if !found {
		return &errors.MissingDependencies{
			MissingDeps: map[string][]string{
				"in-memory-db-secret": {fmt.Sprintf("name: %s, namespace: %s", secretName, app.Namespace)},
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
