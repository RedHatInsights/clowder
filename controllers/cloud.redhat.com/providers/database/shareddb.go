package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"database/sql"

	_ "github.com/lib/pq" // Required to load postgres

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// SharedDBDeployment is the ident referring to the local DB deployment object.
var SharedDBDeployment = rc.NewMultiResourceIdent(ProvName, "shared_db_deployment", &apps.Deployment{})

// SharedDBService is the ident referring to the local DB service object.
var SharedDBService = rc.NewMultiResourceIdent(ProvName, "shared_db_service", &core.Service{})

// SharedDBPVC is the ident referring to the local DB PVC object.
var SharedDBPVC = rc.NewMultiResourceIdent(ProvName, "shared_db_pvc", &core.PersistentVolumeClaim{})

// SharedDBSecret is the ident referring to the local DB secret object.
var SharedDBSecret = rc.NewMultiResourceIdent(ProvName, "shared_db_secret", &core.Secret{})

// SharedDBAppSecret is the ident referring to the shared DB app secret object.
var SharedDBAppSecret = rc.NewSingleResourceIdent(ProvName, "shared_db_app_secret", &core.Secret{})

type sharedDbProvider struct {
	providers.Provider
}

// NewSharedDBProvider returns a new local DB provider object.
func NewSharedDBProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		SharedDBDeployment,
		SharedDBService,
		SharedDBPVC,
		SharedDBSecret,
		SharedDBAppSecret,
	)
	return &sharedDbProvider{Provider: *p}, nil
}

func createVersionedDatabase(p *providers.Provider, version int32) (*config.DatabaseConfig, error) {
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-db-v%s", p.Env.Name, strconv.Itoa(int(version))),
		Namespace: p.Env.Status.TargetNamespace,
	}

	dd := &apps.Deployment{}
	err := p.Cache.Create(SharedDBDeployment, nn, dd)

	if err != nil {
		return nil, err
	}

	dbCfg := config.DatabaseConfig{}

	password, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return nil, errors.Wrap("password generate failed", err)
	}

	pgPassword, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return nil, errors.Wrap("pgPassword generate failed", err)
	}

	dataInit := func() map[string]string {
		return map[string]string{
			"hostname": fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
			"port":     "5432",
			"username": utils.RandString(16),
			"password": password,
			"pgPass":   pgPassword,
			"name":     p.Env.Name,
		}
	}

	secMap, err := providers.MakeOrGetSecret(p.Env, p.Cache, SharedDBSecret, nn, dataInit)
	if err != nil {
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}

	err = dbCfg.Populate(secMap)
	if err != nil {
		return nil, errors.Wrap("couldn't convert to int", err)
	}
	dbCfg.AdminUsername = "postgres"
	dbCfg.SslMode = "disable"

	var image string

	image, err = provutils.GetDefaultDatabaseImage(version)

	if err != nil {
		return nil, err
	}

	imgComponents := strings.Split(image, ":")
	tag := "cyndi-" + imgComponents[1]
	image = imgComponents[0] + ":" + tag

	labels := &map[string]string{"sub": fmt.Sprintf("shared_db_%s", strconv.Itoa(int(version)))}

	provutils.MakeLocalDB(dd, nn, p.Env, labels, &dbCfg, image, p.Env.Spec.Providers.Database.PVC, p.Env.Name, nil)

	if err = p.Cache.Update(SharedDBDeployment, dd); err != nil {
		return nil, err
	}

	s := &core.Service{}
	if err := p.Cache.Create(SharedDBService, nn, s); err != nil {
		return nil, err
	}

	provutils.MakeLocalDBService(s, nn, p.Env, labels)

	if err = p.Cache.Update(SharedDBService, s); err != nil {
		return nil, err
	}

	defaultVolSize := sizing.GetDefaultSizeVol()

	if p.Env.Spec.Providers.Database.PVC {
		pvc := &core.PersistentVolumeClaim{}
		if err := p.Cache.Create(SharedDBPVC, nn, pvc); err != nil {
			return nil, err
		}

		largestDBVolSize := defaultVolSize

		appList, err := p.Env.GetAppsInEnv(p.Ctx, p.Client)
		if err != nil {
			return nil, err
		}
		// Iterate through the list of dbs to find the largest pvc request and
		// use that for its particular version.
		for _, app := range appList.Items {
			// Don't take the largest for a different versioned db
			if version != *app.Spec.Database.Version {
				continue
			}
			dbSize := app.Spec.Database.DBVolumeSize
			if dbSize == "" {
				dbSize = defaultVolSize
			}

			if sizing.IsSizeLarger(dbSize, largestDBVolSize) {
				largestDBVolSize = dbSize
			}

		}

		provutils.MakeLocalDBPVC(pvc, nn, p.Env, sizing.GetVolCapacityForSize(largestDBVolSize))

		if err = p.Cache.Update(SharedDBPVC, pvc); err != nil {
			return nil, err
		}
	}

	return &dbCfg, nil
}

func (db *sharedDbProvider) EnvProvide() error {
	appList, err := db.Env.GetAppsInEnv(db.Ctx, db.Client)
	if err != nil {
		return err
	}

	versionsRequired := map[int32]bool{}

	for _, app := range appList.Items {
		if app.Spec.Database.Name != "" {
			if app.Spec.Database.Version == nil {
				versionsRequired[12] = true
			} else {
				versionsRequired[*app.Spec.Database.Version] = true
			}
		}
	}

	configs := map[int32]*config.DatabaseConfig{}

	for v := range versionsRequired {
		dbCfg, err := createVersionedDatabase(&db.Provider, v)
		if err != nil {
			return err
		}
		configs[v] = dbCfg
	}

	return nil
}

// CreateDatabase ensures a database is created for the given app.  The
// namespaced name passed in must be the actual name of the db resources
func (db *sharedDbProvider) Provide(app *crd.ClowdApp) error {
	if app.Spec.Database.Name == "" && app.Spec.Database.SharedDBAppName == "" {
		return nil
	}

	if app.Spec.Database.SharedDBAppName != "" {
		return db.processSharedDB(app)
	}

	version := int32(12)
	if app.Spec.Database.Version != nil {
		version = *app.Spec.Database.Version
	}

	var dbCfg config.DatabaseConfig

	vSec := &core.Secret{}
	vSecnn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-db-v%s", db.Env.Name, strconv.Itoa(int(version))),
		Namespace: db.Env.Status.TargetNamespace,
	}

	if err := db.Client.Get(db.Ctx, vSecnn, vSec); err != nil {
		return err
	}

	port, err := strconv.Atoi(string(vSec.Data["port"]))
	if err != nil {
		return err
	}

	dbCfg.AdminUsername = "postgres"
	dbCfg.AdminPassword = string(vSec.Data["pgPass"])
	dbCfg.Hostname = string(vSec.Data["hostname"])
	dbCfg.Name = app.Spec.Database.Name
	dbCfg.Password = string(vSec.Data["password"])
	dbCfg.Username = string(vSec.Data["username"])
	dbCfg.Port = int(port)
	dbCfg.SslMode = "disable"

	host := dbCfg.Hostname
	user := dbCfg.AdminUsername
	password := dbCfg.AdminPassword
	dbname := app.Spec.Database.Name

	appSQLConnectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	ctx, cancel := context.WithTimeout(db.Ctx, 5*time.Second)
	defer cancel()

	dbClient, err := sql.Open("postgres", appSQLConnectionString)
	if err != nil {
		return err
	}

	defer dbClient.Close()

	pErr := dbClient.PingContext(ctx)
	if pErr != nil {
		if strings.Contains(pErr.Error(), fmt.Sprintf("database \"%s\" does not exist", app.Spec.Database.Name)) {

			envSQLConnectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, db.Env.Name)

			envDbClient, envErr := sql.Open("postgres", envSQLConnectionString)
			if envErr != nil {
				return envErr
			}

			defer envDbClient.Close()

			sqlStatement := fmt.Sprintf("CREATE DATABASE \"%s\" WITH OWNER=\"%s\";", app.Spec.Database.Name, dbCfg.Username)
			preppedStatement, err := envDbClient.PrepareContext(ctx, sqlStatement)
			if err != nil {
				return err
			}
			_, createErr := preppedStatement.Exec()
			if createErr != nil {
				return createErr
			}
		} else {
			return pErr
		}
	}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-db", app.Name),
		Namespace: app.Namespace,
	}

	secret := &core.Secret{}
	if err := db.Cache.Create(SharedDBAppSecret, nn, secret); err != nil {
		return err
	}

	secret.StringData = map[string]string{
		"hostname": host,
		"port":     "5432",
		"username": dbCfg.Username,
		"password": dbCfg.Password,
		"pgPass":   dbCfg.AdminPassword,
		"name":     app.Spec.Database.Name,
	}

	secret.Name = nn.Name
	secret.Namespace = nn.Namespace
	secret.OwnerReferences = []metav1.OwnerReference{app.MakeOwnerReference()}
	secret.Type = core.SecretTypeOpaque

	if err := db.Cache.Update(SharedDBAppSecret, secret); err != nil {
		return err
	}

	dbCfg.Name = app.Spec.Database.Name
	db.Config.Database = &dbCfg

	return nil
}

func (db *sharedDbProvider) processSharedDB(app *crd.ClowdApp) error {
	err := checkDependency(app)

	if err != nil {
		return err
	}

	dbCfg := config.DatabaseConfig{}
	dbCfg.SslMode = "disable"

	refApp, err := crd.GetAppForDBInSameEnv(db.Ctx, db.Client, app, false)

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

	err = dbCfg.Populate(&secMap)
	if err != nil {
		return errors.Wrap("couldn't convert to int", err)
	}
	dbCfg.AdminUsername = "postgres"

	db.Config.Database = &dbCfg

	return nil
}
