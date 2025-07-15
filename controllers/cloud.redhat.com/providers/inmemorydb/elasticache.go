package inmemorydb

import (
	"fmt"
	"strconv"

	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
)

type elasticache struct {
	providers.Provider
}

func (e *elasticache) EnvProvide() error {
	return nil
}

func (e *elasticache) Provide(app *crd.ClowdApp) error {
	var refApp *crd.ClowdApp
	var ecNameSpace string

	if !app.Spec.InMemoryDB {
		return nil
	}

	secretName := "in-memory-db"
	secrets := core.SecretList{}

	if app.Spec.SharedInMemoryDBAppName != "" {
		err := checkDependency(app)

		if err != nil {
			return err
		}

		refApp, err = crd.GetAppForDBInSameEnv(e.Provider.Ctx, e.Provider.Client, app, true)

		if err != nil {
			return err
		}

		ecNameSpace = refApp.Namespace
	} else {
		ecNameSpace = app.Namespace
	}

	err := e.Provider.Client.List(e.Provider.Ctx, &secrets, client.InNamespace(ecNameSpace))

	if err != nil {
		msg := fmt.Sprintf("Failed to list secrets in %s", ecNameSpace)
		return errors.Wrap(msg, err)
	}

	found := false
	sslmode := true

	creds := config.InMemoryDBConfig{}

	creds.SslMode = &sslmode

	for _, secret := range secrets.Items {
		if secret.Name == secretName {
			port, err := strconv.Atoi(string(secret.Data["db.port"]))

			if err != nil {
				return errors.Wrap(
					fmt.Sprintf("failed to parse port from secret '%s' in namespace '%s'", secretName, ecNameSpace),
					err,
				)
			}

			// ElastiCache and Terraform resources, via qontract-reconcile, guarantee that `db.auth_token` is provided
			// only if in-transit encryption is enabled.
			passwd := string(secret.Data["db.auth_token"])
			if passwd != "" {
				creds.Password = &passwd
			}

			creds.Hostname = string(secret.Data["db.endpoint"])
			creds.Port = int(port)
			found = true
			break
		}
	}

	if !found {
		missingDeps := errors.MakeMissingDependencies(errors.MissingDependency{
			Source:  "inmemorydb",
			Details: fmt.Sprintf("No inmemorydb secret named '%s' found in namespace '%s'", secretName, ecNameSpace),
		})
		return &missingDeps
	}

	e.Provider.Config.InMemoryDb = &creds

	return nil
}

// NewElasticache returns a new elasticache provider object.
func NewElasticache(p *providers.Provider) (providers.ClowderProvider, error) {
	return &elasticache{Provider: *p}, nil
}
