package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing/sizingconfig"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// SingleDBDeployment is the ident referring to the single local DB deployment object
var SingleDBDeployment = rc.NewSingleResourceIdent(ProvName, "single_db_deployment", &apps.Deployment{})

// SingleDBScret is the ident referring to the single local DB secret object
var SingleDBSecret = rc.NewMultiResourceIdent(ProvName, "single_db_secret", &core.Secret{})

// SingleDBPVC is the ident referring to the single local DB PersitentVolumeClaim
var SingleDBPVC = rc.NewSingleResourceIdent(ProvName, "single_db_pvc", &core.PersistentVolumeClaim{})

// SingleDBService is the ident referring to the single local DB service object.
var SingleDBService = rc.NewMultiResourceIdent(ProvName, "single_db_service", &core.Service{})

func NewSingleDBProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		SingleDBDeployment,
		SingleDBSecret,
		SingleDBPVC,
		SingleDBService,
	)
	return &singleDbProvider{Provider: *p}, nil
}

type singleDbProvider struct {
	providers.Provider
}

func (db *singleDbProvider) DBNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-db-single", db.Env.Name),
		Namespace: db.Env.Status.TargetNamespace,
	}
}

func (db *singleDbProvider) EnvProvide() error {
	appList, err := db.Env.GetAppsInEnv(db.Ctx, db.Client)
	if err != nil {
		return err
	}

	needsDb := false
	for _, app := range appList.Items {
		if app.Spec.Database.Name != "" {
			needsDb = true
			break
		}
	}

	if needsDb {
		if _, err := db.createDatabaseDeployment(); err != nil {
			return err
		}
		if db.Env.Spec.Providers.Database.PVC {
			if err := db.createDatabasePVC(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Creates a single database deployment locked to one version with a main secret
func (db *singleDbProvider) createDatabaseDeployment() (*config.DatabaseConfig, error) {
	nn := db.DBNamespacedName()

	dd := &apps.Deployment{}
	ownerrefs := []metav1.OwnerReference{db.Env.MakeOwnerReference()}
	dd.ObjectMeta.OwnerReferences = ownerrefs
	err := db.Cache.Create(SingleDBDeployment, nn, dd)
	if err != nil {
		return nil, err
	}

	dbName := db.Env.Name
	dbCfg, err := db.createOrReadDbConfig(db.Env, nn, dbName, true)
	if err != nil {
		return nil, err
	}

	labels := &map[string]string{
		"sub": "single_db",
	}
	usePVC := db.Env.Spec.Providers.Database.PVC
	provutils.MakeLocalDB(dd, nn, db.Env, labels, dbCfg, provutils.DefaultImageDatabasePG, usePVC, dbName, nil)
	if err := db.Cache.Update(SingleDBDeployment, dd); err != nil {
		return dbCfg, err
	}

	svc := &core.Service{}
	svc.ObjectMeta.OwnerReferences = ownerrefs
	if err := db.Cache.Create(SingleDBService, nn, svc); err != nil {
		return dbCfg, err
	}

	provutils.MakeLocalDBService(svc, nn, db.Env, labels)

	if err = db.Cache.Update(SingleDBService, svc); err != nil {
		return dbCfg, err
	}

	return dbCfg, nil
}

func (db *singleDbProvider) createDatabasePVC() error {
	nn := db.DBNamespacedName()

	pvc := &core.PersistentVolumeClaim{}
	if err := db.Cache.Create(SingleDBPVC, nn, pvc); err != nil {
		return err
	}

	// TODO handle volume capacity
	capacity := sizing.GetVolCapacityForSize(sizingconfig.SizeLarge)
	provutils.MakeLocalDBPVC(pvc, nn, db.Env, capacity)
	if err := db.Cache.Update(SingleDBPVC, pvc); err != nil {
		return err
	}
	return nil
}

func (db *singleDbProvider) Provide(app *crd.ClowdApp) error {
	if app.Spec.Database.SharedDBAppName != "" {
		return db.processSharedDB(app)
	} else if app.Spec.Database.Name != "" {
		return db.provideAppDB(app)
	}
	return nil
}

func (db *singleDbProvider) provideAppDB(app *crd.ClowdApp) error {
	dbnn := db.DBNamespacedName()
	dbCfg := &config.DatabaseConfig{}

	if err := provutils.ReadDbConfigFromSecret(db.Provider, SingleDBSecret, dbCfg, dbnn); err != nil {
		return err
	}

	// Create database config and secret for the app,
	// without the admin credentials.
	appnn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-db", app.Name),
		Namespace: app.Namespace,
	}
	appDbCfg, err := db.createOrReadDbConfig(app, appnn, app.Name, false)
	if err != nil {
		return err
	}
	db.Config.Database = appDbCfg

	// reconcile access
	if err := db.reconcileDBAppAccess(dbCfg, appDbCfg); err != nil {
		return err
	}

	return nil
}

func (db *singleDbProvider) processSharedDB(app *crd.ClowdApp) error {
	refApp, err := crd.GetAppForDBInSameEnv(db.Ctx, db.Client, app)
	if err != nil {
		return err
	}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-db", refApp.Name),
		Namespace: refApp.Namespace,
	}
	dbCfg := &config.DatabaseConfig{}
	if err := provutils.ReadDbConfigFromSecret(db.Provider, SingleDBSecret, dbCfg, nn); err != nil {
		return errors.Wrap(fmt.Sprintf("sharedDBApp %s", refApp.Name), err)
	}

	db.Config.Database = dbCfg
	return nil
}

func (db *singleDbProvider) Hostname() string {
	nn := db.DBNamespacedName()
	return fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace)
}

func (db *singleDbProvider) createOrReadDbConfig(obj obj.ClowdObject, nn types.NamespacedName, username string, setAdmin bool) (cfg *config.DatabaseConfig, err error) {
	cfg = &config.DatabaseConfig{}
	password, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return cfg, errors.Wrap("password generate failed", err)
	}

	var pgPassword string
	if setAdmin {
		pgPassword, err = utils.RandPassword(16, provutils.RCharSet)
		if err != nil {
			return cfg, errors.Wrap("pgPassword generate failed", err)
		}
	}

	dataInit := func() map[string]string {
		return map[string]string{
			"hostname": db.Hostname(),
			"port":     provutils.DefaultPGPort,
			"name":     db.Env.Name,
			"username": username,
			"password": password,
			"pgPass":   pgPassword,
		}
	}

	var secMap *map[string]string
	secMap, err = providers.MakeOrGetSecret(obj, db.Cache, SingleDBSecret, nn, dataInit)
	if err != nil {
		return cfg, errors.Wrap("couldn't set/get secret", err)
	}

	err = cfg.Populate(secMap)
	if err != nil {
		return cfg, errors.Wrap("couldn't populate db config from secret", err)
	}
	if setAdmin {
		cfg.AdminUsername = provutils.DefaultPGAdminUsername
	}
	cfg.SslMode = "disable"
	return
}

func (db *singleDbProvider) reconcileDBAppAccess(envCfg *config.DatabaseConfig, appCfg *config.DatabaseConfig) error {
	appSQLConnectionString := provutils.PGAdminConnectionStr(envCfg, envCfg.Name)

	ctx, cancel := context.WithTimeout(db.Ctx, 5*time.Second)
	defer cancel()

	dbClient, err := sql.Open("postgres", appSQLConnectionString)
	if err != nil {
		return errors.Wrap("unable ,to connect to db", err)
	}
	defer dbClient.Close()

	if err := dbClient.PingContext(ctx); err != nil {
		return err
	}

	username := appCfg.Username
	password := appCfg.Password
	rows, err := dbClient.QueryContext(ctx, "SELECT TRUE FROM pg_roles WHERE rolname = $1;", username)
	if err != nil {
		return errors.Wrap("unable to query for roles", err)
	}

	var roleExists = rows.Next()
	rows.Close()

	if roleExists {
		_, err = dbClient.ExecContext(ctx,
			fmt.Sprintf("ALTER ROLE %s WITH LOGIN ENCRYPTED PASSWORD %s;",
				pq.QuoteIdentifier(username), pq.QuoteLiteral(password),
			))
	} else {
		_, err = dbClient.ExecContext(ctx,
			fmt.Sprintf("CREATE ROLE %s WITH LOGIN ENCRYPTED PASSWORD %s;",
				pq.QuoteIdentifier(username), pq.QuoteLiteral(password),
			))
	}
	if err != nil {
		return errors.Wrap("unable to create/alter role", err)
	}

	_, err = dbClient.ExecContext(ctx,
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s;",
			pq.QuoteIdentifier(username), pq.QuoteIdentifier(username),
		))
	if err != nil {
		return errors.Wrap("unable to create db schema for the app", err)
	}

	return nil
}
