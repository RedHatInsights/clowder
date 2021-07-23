package database

import (
	"fmt"
	"strings"

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

// LocalDBDeployment is the ident refering to the local DB deployment object.
var LocalDBDeployment = providers.NewSingleResourceIdent(ProvName, "local_db_deployment", &apps.Deployment{})

// LocalDBService is the ident refering to the local DB service object.
var LocalDBService = providers.NewSingleResourceIdent(ProvName, "local_db_service", &core.Service{})

// LocalDBPVC is the ident refering to the local DB PVC object.
var LocalDBPVC = providers.NewSingleResourceIdent(ProvName, "local_db_pvc", &core.PersistentVolumeClaim{})

// LocalDBSecret is the ident refering to the local DB secret object.
var LocalDBSecret = providers.NewSingleResourceIdent(ProvName, "local_db_secret", &core.Secret{})

type localDbProvider struct {
	providers.Provider
	Config config.DatabaseConfig
}

// NewLocalDBProvider returns a new local DB provider object.
func NewLocalDBProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &localDbProvider{Provider: *p}, nil
}

// CreateDatabase ensures a database is created for the given app.  The
// namespaced name passed in must be the actual name of the db resources
func (db *localDbProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if app.Spec.Database.Name == "" && app.Spec.Database.SharedDBAppName == "" {
		return nil
	}

	if app.Spec.Database.SharedDBAppName != "" {
		return db.processSharedDB(app, c)
	}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-db", app.Name),
		Namespace: app.Namespace,
	}

	dd := &apps.Deployment{}
	err := db.Cache.Create(LocalDBDeployment, nn, dd)

	if err != nil {
		return err
	}

	dbCfg := config.DatabaseConfig{}
	dataInit := func() map[string]string {
		return map[string]string{
			"hostname": fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
			"port":     "5432",
			"username": utils.RandString(16),
			"password": utils.RandString(16),
			"pgPass":   utils.RandString(16),
			"name":     app.Spec.Database.Name,
		}
	}

	secMap, err := providers.MakeOrGetSecret(db.Ctx, app, db.Cache, LocalDBSecret, nn, dataInit)
	if err != nil {
		return errors.Wrap("Couldn't set/get secret", err)
	}

	dbCfg.Populate(secMap)
	dbCfg.AdminUsername = "postgres"
	dbCfg.SslMode = "disable"

	db.Config = dbCfg

	var image string

	var dbVersion int32 = 12
	if app.Spec.Database.Version != nil {
		dbVersion = *(app.Spec.Database.Version)
	}

	image, ok := imageList[dbVersion]

	if !ok {
		return errors.New(fmt.Sprintf("Requested image version (%v), doesn't exist", dbVersion))
	}

	if app.Spec.Cyndi.Enabled {
		imgComponents := strings.Split(image, ":")
		tag := "cyndi-" + imgComponents[1]
		image = imgComponents[0] + ":" + tag
	}

	provutils.MakeLocalDB(dd, nn, app, &dbCfg, image, db.Env.Spec.Providers.Database.PVC, app.Spec.Database.Name)

	if err = db.Cache.Update(LocalDBDeployment, dd); err != nil {
		return err
	}

	s := &core.Service{}
	if err := db.Cache.Create(LocalDBService, nn, s); err != nil {
		return err
	}

	provutils.MakeLocalDBService(s, nn, app)

	if err = db.Cache.Update(LocalDBService, s); err != nil {
		return err
	}

	if db.Env.Spec.Providers.Database.PVC {
		pvc := &core.PersistentVolumeClaim{}
		if err := db.Cache.Create(LocalDBPVC, nn, pvc); err != nil {
			return err
		}

		provutils.MakeLocalDBPVC(pvc, nn, app)

		if err = db.Cache.Update(LocalDBPVC, pvc); err != nil {
			return err
		}
	}
	c.Database = &db.Config
	return nil
}

func (db *localDbProvider) processSharedDB(app *crd.ClowdApp, c *config.AppConfig) error {
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
