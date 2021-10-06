package database

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var rdsCa string

const caURL string = "https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem"

type appInterface struct {
	providers.Provider
	Config config.DatabaseConfigContainer
}

func fetchCa() (string, error) {
	resp, err := http.Get(caURL)

	if err != nil {
		return "", errors.Wrap("Error fetching CA bundle", err)
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Bad status code: %d", resp.StatusCode)
		return "", errors.New(msg)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		errors.Wrap("Error reading response body", err)
	}

	caBundle := string(body)

	if !strings.HasPrefix(caBundle, "-----BEGIN CERTIFICATE") {
		return "", errors.New("Invalid RDS CA bundle")
	}

	return caBundle, nil
}

// NewAppInterfaceDBProvider creates a new app-interface DB provider obejct.
func NewAppInterfaceDBProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	provider := appInterface{Provider: *p}

	if rdsCa == "" {
		_rdsCa, err := fetchCa()

		if err != nil {
			return nil, errors.Wrap("Failed to fetch RDS CA bundle", err)
		}

		rdsCa = _rdsCa
	}
	return &provider, nil
}

func (a *appInterface) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if app.Spec.Database.Name == "" && app.Spec.Database.SharedDBAppName == "" {
		return nil
	}

	if app.Spec.Database.Name != "" && app.Spec.Database.SharedDBAppName != "" {
		return errors.New("Cannot set dbName & shared db app name")
	}

	var dbSpec crd.DatabaseSpec
	var namespace string
	var searchAppName string

	if app.Spec.Database.Name != "" {
		dbSpec = app.Spec.Database
		namespace = app.Namespace
		searchAppName = app.Name
	} else if app.Spec.Database.SharedDBAppName != "" {
		err := checkDependency(app)
		if err != nil {
			return err
		}

		refApp, err := crd.GetAppForDBInSameEnv(a.Ctx, a.Client, app)

		if err != nil {
			return err
		}

		dbSpec = refApp.Spec.Database
		namespace = refApp.Namespace
		searchAppName = refApp.Name
	}

	matched, err := GetDbConfig(a.Ctx, a.Client, namespace, app.Name, searchAppName, dbSpec)

	if err != nil {
		return err
	}

	a.Config = *matched

	c.Database = &a.Config.Config

	return nil
}

func GetDbConfig(ctx context.Context, pClient client.Client, namespace, appName, searchAppName string, dbSpec crd.DatabaseSpec) (*config.DatabaseConfigContainer, error) {
	secrets := core.SecretList{}
	err := pClient.List(ctx, &secrets, client.InNamespace(namespace))

	if err != nil {
		msg := fmt.Sprintf("Failed to list secrets in %s", namespace)
		return nil, errors.Wrap(msg, err)
	}

	sort.Slice(secrets.Items, func(i, j int) bool {
		return secrets.Items[i].Name < secrets.Items[j].Name
	})

	var matched config.DatabaseConfigContainer

	matches, err := searchAnnotationSecret(searchAppName, secrets.Items)

	if err != nil {
		return nil, errors.Wrap("failed to extract annotated secret", err)
	}

	if len(matches) == 0 {

		dbConfigs, err := genDbConfigs(secrets.Items)

		if err != nil {
			return nil, err
		}

		matched = resolveDb(dbSpec, dbConfigs)

		if matched == (config.DatabaseConfigContainer{}) {
			return nil, &errors.MissingDependencies{
				MissingDeps: map[string][]string{
					"database": {appName},
				},
			}
		}
	} else {
		matched = matches[0]
	}

	// The creds given by app-interface have elevated privileges
	matched.Config.AdminPassword = matched.Config.Password
	matched.Config.AdminUsername = matched.Config.Username
	matched.Config.RdsCa = &rdsCa

	return &matched, nil

}

func resolveDb(spec crd.DatabaseSpec, c []config.DatabaseConfigContainer) config.DatabaseConfigContainer {
	for _, config := range c {
		hostname := strings.Split(config.Config.Hostname, ".")[0]
		nameSegments := strings.Split(hostname, "-")
		segLen := len(nameSegments)
		lastSegment := nameSegments[segLen-1]

		if lastSegment == "readonly" {
			continue
		}

		dbName := strings.Join(nameSegments[:segLen-1], "-")

		if dbName == spec.Name {
			return config
		}
	}

	return config.DatabaseConfigContainer{}
}

func genDbConfigs(secrets []core.Secret) ([]config.DatabaseConfigContainer, error) {
	configs := []config.DatabaseConfigContainer{}

	var err error

	extractFn := func(secret *core.Secret) {
		port, erro := strconv.Atoi(string(secret.Data["db.port"]))

		if erro != nil {
			err = errors.Wrap("Failed to parse DB port", err)
			return
		}

		dbConfig := config.DatabaseConfigContainer{
			Config: config.DatabaseConfig{
				Hostname: string(secret.Data["db.host"]),
				Port:     port,
				Username: string(secret.Data["db.user"]),
				Password: string(secret.Data["db.password"]),
				Name:     string(secret.Data["db.name"]),
				SslMode:  "verify-full",
			},
			Ref: types.NamespacedName{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		}

		configs = append(configs, dbConfig)
	}

	keys := []string{"db.host", "db.port", "db.user", "db.password", "db.name"}
	providers.ExtractSecretData(secrets, extractFn, keys...)

	if err != nil {
		return nil, err
	}

	return configs, nil
}

func searchAnnotationSecret(appName string, secrets []core.Secret) ([]config.DatabaseConfigContainer, error) {
	for _, secret := range secrets {
		anno := secret.GetAnnotations()
		if v, ok := anno["clowder/database"]; ok && v == appName {
			configs, err := genDbConfigs([]core.Secret{secret})
			return configs, err
		}
	}
	return []config.DatabaseConfigContainer{}, nil
}
