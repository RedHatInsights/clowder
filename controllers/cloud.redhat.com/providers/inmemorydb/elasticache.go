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

func (e *elasticache) Provide(app *crd.ClowdApp, appConfig *config.AppConfig) error {
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

	creds := config.InMemoryDBConfig{}

	for _, secret := range secrets.Items {
		if secret.Name == secretName {
			port, err := strconv.Atoi(string(secret.Data["db.port"]))

			if err != nil {
				return errors.Wrap(
					fmt.Sprintf("failed to parse port from secret '%s' in namespace '%s'", secretName, app.Namespace),
					err,
				)
			}

			passwd := string(secret.Data["db.auth_token"])
			if passwd != "" {
				creds.Password = &passwd
			}

			creds.Hostname = string(secret.Data["db.endpoint"])
			creds.Port = port
			found = true
			break
		}
	}

	if !found {
		missingDeps := errors.MakeMissingDependencies(errors.MissingDependency{
			Source:  "inmemorydb",
			Details: fmt.Sprintf("No inmemorydb secret named '%s' found in namespace '%s'", secretName, app.Namespace),
		})
		return &missingDeps
	}

	appConfig.InMemoryDb = &creds

	return nil
}

// NewElasticache returns a new elasticache provider object.
func NewElasticache(p *providers.Provider) (providers.ClowderProvider, error) {
	config := config.InMemoryDBConfig{}

	redisProvider := elasticache{Provider: *p, Config: config}

	return &redisProvider, nil
}
