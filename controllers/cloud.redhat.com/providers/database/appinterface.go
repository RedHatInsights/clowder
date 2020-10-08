package database

import (
	"fmt"
	"strconv"
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type appInterface struct {
	p.Provider
	Config []config.DatabaseConfig
}

func (a *appInterface) Configure(c *config.AppConfig) {
	c.Database = a.Config
}

func NewAppInterfaceObjectstore(p *p.Provider) (DatabaseProvider, error) {
	provider := appInterface{Provider: *p}

	return &provider, nil
}

func (a *appInterface) CreateDatabase(app *crd.ClowdApp) error {
	if len(app.Spec.Database) == 0 {
		return nil
	}

	secrets := core.SecretList{}
	err := a.Client.List(a.Ctx, &secrets, client.InNamespace(app.Namespace))

	if err != nil {
		msg := fmt.Sprintf("Failed to list secrets in %s", app.Namespace)
		return errors.Wrap(msg, err)
	}

	dbConfigs, err := genDbConfigs(secrets.Items)

	if err != nil {
		return err
	}

	matched, missing := resolveDb(app.Spec.Database, dbConfigs)

	if len(missing) > 0 {
		return &errors.MissingDependencies{
			MissingDeps: map[string][]string{
				"database": missing,
			},
		}
	}

	a.Config = matched
	return nil
}

func resolveDb(specs []crd.InsightsDatabaseSpec, c []config.DatabaseConfig) ([]config.DatabaseConfig, []string) {
	missing := []string{}
	matched := []config.DatabaseConfig{}
	for _, spec := range specs {
		found := false
		for _, config := range c {
			suffixSegments := 1 // environment
			hostname := strings.Split(config.Hostname, ".")[0]
			nameSegments := strings.Split(hostname, "-")
			segLen := len(nameSegments)
			lastSegment := nameSegments[segLen-1]

			if lastSegment == "readonly" {
				if !spec.Readonly {
					continue
				}
				suffixSegments = 2
			} else if spec.Readonly {
				continue
			}

			dbName := strings.Join(nameSegments[:segLen-suffixSegments], "-")

			if dbName == spec.Name {
				found = true
				matched = append(matched, config)
			}
		}

		if !found {
			name := spec.Name

			if spec.Readonly {
				name = name + " (readonly)"
			}

			missing = append(missing, name)
		}
	}

	return matched, missing
}

func genDbConfigs(secrets []core.Secret) ([]config.DatabaseConfig, error) {
	configs := []config.DatabaseConfig{}

	var err error

	extractFn := func(m map[string][]byte) {
		port, erro := strconv.Atoi(string(m["db.port"]))

		if erro != nil {
			err = errors.Wrap("Failed to parse DB port", err)
			return
		}

		dbConfig := config.DatabaseConfig{
			Hostname: string(m["db.host"]),
			Port:     port,
			Username: string(m["db.user"]),
			Password: string(m["db.password"]),
			Name:     string(m["db.name"]),
		}

		configs = append(configs, dbConfig)
	}

	keys := []string{"db.host", "db.port", "db.user", "db.password", "db.name"}
	p.ExtractSecretData(secrets, extractFn, keys...)

	if err != nil {
		return nil, err
	}

	return configs, nil
}
