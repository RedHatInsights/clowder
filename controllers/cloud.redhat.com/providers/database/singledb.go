package database

import (
	"fmt"
	"strings"

	"database/sql"

	_ "github.com/lib/pq"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// SingleDBDeployment is the ident referring to the local DB deployment object.
var SingleDBDeployment = providers.NewSingleResourceIdent(ProvName, "single_db_deployment", &apps.Deployment{})

// SingleDBService is the ident referring to the local DB service object.
var SingleDBService = providers.NewSingleResourceIdent(ProvName, "single_db_service", &core.Service{})

// SingleDBPVC is the ident referring to the local DB PVC object.
var SingleDBPVC = providers.NewSingleResourceIdent(ProvName, "single_db_pvc", &core.PersistentVolumeClaim{})

// SingleDBSecret is the ident referring to the local DB secret object.
var SingleDBSecret = providers.NewSingleResourceIdent(ProvName, "single_db_secret", &core.Secret{})

type singleDbProvider struct {
	providers.Provider
	Config config.DatabaseConfig
}

// NewSingleDBProvider returns a new local DB provider object.
func NewSingleDBProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	dbp := &singleDbProvider{Provider: *p, Config: config.DatabaseConfig{}}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-db", p.Env.Name),
		Namespace: p.Env.Status.TargetNamespace,
	}

	dd := &apps.Deployment{}
	err := p.Cache.Create(SingleDBDeployment, nn, dd)

	if err != nil {
		return &singleDbProvider{}, err
	}

	dbCfg := config.DatabaseConfig{}
	dataInit := func() map[string]string {
		return map[string]string{
			"hostname": fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
			"port":     "5432",
			"username": utils.RandString(16),
			"password": utils.RandString(16),
			"pgPass":   utils.RandString(16),
			"name":     p.Env.Name,
		}
	}

	secMap, err := providers.MakeOrGetSecret(p.Ctx, p.Env, p.Cache, SingleDBSecret, nn, dataInit)
	if err != nil {
		return &singleDbProvider{}, errors.Wrap("Couldn't set/get secret", err)
	}

	dbCfg.Populate(secMap)
	dbCfg.AdminUsername = "postgres"
	dbCfg.SslMode = "disable"

	dbp.Config = dbCfg

	var image string

	var dbVersion int32 = 12
	// if app.Spec.Database.Version != nil {
	// 	dbVersion = *(app.Spec.Database.Version)
	// }

	image, ok := imageList[dbVersion]

	if !ok {
		return &singleDbProvider{}, errors.New(fmt.Sprintf("Requested image version (%v), doesn't exist", dbVersion))
	}

	imgComponents := strings.Split(image, ":")
	tag := "cyndi-" + imgComponents[1]
	image = imgComponents[0] + ":" + tag

	labels := &map[string]string{"sub": "single_db"}

	provutils.MakeLocalDB(dd, nn, p.Env, labels, &dbCfg, image, p.Env.Spec.Providers.Database.PVC, p.Env.Name)

	if err = p.Cache.Update(SingleDBDeployment, dd); err != nil {
		return &singleDbProvider{}, err
	}

	s := &core.Service{}
	if err := p.Cache.Create(SingleDBService, nn, s); err != nil {
		return &singleDbProvider{}, err
	}

	provutils.MakeLocalDBService(s, nn, p.Env, labels)

	if err = p.Cache.Update(SingleDBService, s); err != nil {
		return &singleDbProvider{}, err
	}

	if p.Env.Spec.Providers.Database.PVC {
		pvc := &core.PersistentVolumeClaim{}
		if err := p.Cache.Create(SingleDBPVC, nn, pvc); err != nil {
			return &singleDbProvider{}, err
		}

		provutils.MakeLocalDBPVC(pvc, nn, p.Env)

		if err = p.Cache.Update(SingleDBPVC, pvc); err != nil {
			return &singleDbProvider{}, err
		}
	}

	return &singleDbProvider{Provider: *p, Config: dbCfg}, nil
}

// CreateDatabase ensures a database is created for the given app.  The
// namespaced name passed in must be the actual name of the db resources
func (db *singleDbProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if app.Spec.Database.Name == "" && app.Spec.Database.SharedDBAppName == "" {
		return nil
	}

	if app.Spec.Database.SharedDBAppName != "" {
		return db.processSharedDB(app, c)
	}

	host := db.Config.Hostname
	port := db.Config.Port
	user := db.Config.AdminUsername
	password := db.Config.AdminPassword
	dbname := app.Spec.Database.Name

	appSqlConnectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	dbClient, err := sql.Open("postgres", appSqlConnectionString)
	if err != nil {
		return err
	}

	defer dbClient.Close()

	pErr := dbClient.Ping()
	if pErr != nil {
		if strings.Contains(pErr.Error(), fmt.Sprintf("database \"%s\" does not exist", app.Spec.Database.Name)) {

			envSqlConnectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, db.Env.Name)

			envDbClient, envErr := sql.Open("postgres", envSqlConnectionString)
			if envErr != nil {
				return envErr
			}

			defer envDbClient.Close()

			sqlStatement := fmt.Sprintf("CREATE DATABASE \"%s\" WITH OWNER=\"%s\";", app.Spec.Database.Name, db.Config.Username)
			_, createErr := envDbClient.Exec(sqlStatement)
			if createErr != nil {
				return createErr
			}
		} else {
			return pErr
		}
	}

	c.Database = &db.Config

	return nil
}

func (db *singleDbProvider) processSharedDB(app *crd.ClowdApp, c *config.AppConfig) error {
	err := checkDependency(app)

	if err != nil {
		return err
	}

	dbCfg := config.DatabaseConfig{}
	dbCfg.SslMode = "disable"

	refApp, err := crd.GetAppForDBInSameEnv(db.Ctx, db.Client, app)

	if err != nil {
		return err
	}

	secret := core.Secret{}

	inn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-db", refApp.Name),
		Namespace: refApp.Namespace,
	}

	// This is a REAL call here, not a cached call as the reconciliation must have been processed
	// for the app we depend on.
	if err = db.Client.Get(db.Ctx, inn, &secret); err != nil {
		return errors.Wrap("Couldn't set/get secret", err)
	}

	secMap := make(map[string]string)

	for k, v := range secret.Data {
		(secMap)[k] = string(v)
	}

	dbCfg.Populate(&secMap)
	dbCfg.AdminUsername = "postgres"

	db.Config = dbCfg
	c.Database = &db.Config

	return nil
}
