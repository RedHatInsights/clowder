package database

import (
	"fmt"
	"strconv"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// SingleDBDeployment is the ident referring to the local DB deployment object.
var SingleDBDeployment = providers.NewMultiResourceIdent(ProvName, "single_db_deployment", &apps.Deployment{})

// SingleDBService is the ident referring to the local DB service object.
var SingleDBService = providers.NewMultiResourceIdent(ProvName, "single_db_service", &core.Service{})

// SingleDBPVC is the ident referring to the local DB PVC object.
var SingleDBPVC = providers.NewMultiResourceIdent(ProvName, "single_db_pvc", &core.PersistentVolumeClaim{})

// SingleDBSecret is the ident referring to the local DB secret object.
var SingleDBSecret = providers.NewMultiResourceIdent(ProvName, "single_db_secret", &core.Secret{})

// SingleDBSecret is the ident referring to the local DB secret object.
var SingleDBAppSecret = providers.NewSingleResourceIdent(ProvName, "single_db_app_secret", &core.Secret{})

type singleDbProvider struct {
	providers.Provider
	Config map[int32]*config.DatabaseConfig
}

// NewSingleDBProvider returns a new local DB provider object.
func NewSingleDBProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	appList, err := p.Env.GetAppsInEnv(p.Ctx, p.Client)
	if err != nil {
		return nil, err
	}

	versionsRequired := map[int32]bool{}

	for _, app := range appList.Items {
		if app.Spec.Database.Name != "" {
			fmt.Printf("\n----%v----\n", app.Spec.Database.Version)
			if app.Spec.Database.Version == nil {
				versionsRequired[12] = true
			} else {
				versionsRequired[*app.Spec.Database.Version] = true
			}
		}
	}

	configs := map[int32]*config.DatabaseConfig{}

	for v, _ := range versionsRequired {
		dbCfg, err := createVersionedDatabase(p, v)
		if err != nil {
			return nil, err
		}
		configs[v] = dbCfg
	}

	return &singleDbProvider{Provider: *p, Config: configs}, nil
}

func createVersionedDatabase(p *providers.Provider, version int32) (*config.DatabaseConfig, error) {
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-db-v%s", p.Env.Name, strconv.Itoa(int(version))),
		Namespace: p.Env.Status.TargetNamespace,
	}

	dd := &apps.Deployment{}
	err := p.Cache.Create(SingleDBDeployment, nn, dd)

	if err != nil {
		return nil, err
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
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}

	dbCfg.Populate(secMap)
	dbCfg.AdminUsername = "postgres"
	dbCfg.SslMode = "disable"

	var image string

	image, ok := imageList[version]

	if !ok {
		return nil, errors.New(fmt.Sprintf("Requested image version (%v), doesn't exist", version))
	}

	imgComponents := strings.Split(image, ":")
	tag := "cyndi-" + imgComponents[1]
	image = imgComponents[0] + ":" + tag

	labels := &map[string]string{"sub": fmt.Sprintf("single_db_%s", strconv.Itoa(int(version)))}

	provutils.MakeLocalDB(dd, nn, p.Env, labels, &dbCfg, image, p.Env.Spec.Providers.Database.PVC, p.Env.Name)

	if err = p.Cache.Update(SingleDBDeployment, dd); err != nil {
		return nil, err
	}

	s := &core.Service{}
	if err := p.Cache.Create(SingleDBService, nn, s); err != nil {
		return nil, err
	}

	provutils.MakeLocalDBService(s, nn, p.Env, labels)

	if err = p.Cache.Update(SingleDBService, s); err != nil {
		return nil, err
	}

	if p.Env.Spec.Providers.Database.PVC {
		pvc := &core.PersistentVolumeClaim{}
		if err := p.Cache.Create(SingleDBPVC, nn, pvc); err != nil {
			return nil, err
		}

		provutils.MakeLocalDBPVC(pvc, nn, p.Env)

		if err = p.Cache.Update(SingleDBPVC, pvc); err != nil {
			return nil, err
		}
	}

	return &dbCfg, nil
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

	version := int32(12)
	if app.Spec.Database.Version != nil {
		version = *app.Spec.Database.Version
	}

	dbCfg := db.Config[version]

	host := dbCfg.Hostname
	port := dbCfg.Port
	user := dbCfg.AdminUsername
	password := dbCfg.AdminPassword
	dbname := app.Spec.Database.Name

	appSqlConnectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	fmt.Printf("\n%v\n", appSqlConnectionString)
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

			sqlStatement := fmt.Sprintf("CREATE DATABASE \"%s\" WITH OWNER=\"%s\";", app.Spec.Database.Name, dbCfg.Username)
			_, createErr := envDbClient.Exec(sqlStatement)
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
	if err := db.Cache.Create(SingleDBAppSecret, nn, secret); err != nil {
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
	secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{app.MakeOwnerReference()}
	secret.Type = core.SecretTypeOpaque

	if err := db.Cache.Update(SingleDBAppSecret, secret); err != nil {
		return err
	}

	dbCfg.Name = app.Spec.Database.Name
	c.Database = dbCfg

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

	c.Database = &dbCfg

	return nil
}
