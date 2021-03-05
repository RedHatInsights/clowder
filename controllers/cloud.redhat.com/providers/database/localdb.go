package database

import (
	"fmt"
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	provutils "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/utils"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type localDbProvider struct {
	p.Provider
	Config config.DatabaseConfig
}

// NewLocalDBProvider returns a new local DB provider object.
func NewLocalDBProvider(p *p.Provider) (providers.ClowderProvider, error) {
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

	dd := apps.Deployment{}
	exists, err := utils.UpdateOrErr(db.Client.Get(db.Ctx, nn, &dd))

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

	secMap, err := config.MakeOrGetSecret(db.Ctx, app, db.Client, nn, dataInit)
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

	provutils.MakeLocalDB(&dd, nn, app, &dbCfg, image, db.Env.Spec.Providers.Database.PVC, app.Spec.Database.Name)

	if err = exists.Apply(db.Ctx, db.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	update, err := utils.UpdateOrErr(db.Client.Get(db.Ctx, nn, &s))

	if err != nil {
		return err
	}

	provutils.MakeLocalDBService(&s, nn, app)

	if err = update.Apply(db.Ctx, db.Client, &s); err != nil {
		return err
	}

	if db.Env.Spec.Providers.Database.PVC {
		pvc := core.PersistentVolumeClaim{}
		update, err = utils.UpdateOrErr(db.Client.Get(db.Ctx, nn, &pvc))

		if err != nil {
			return err
		}

		provutils.MakeLocalDBPVC(&pvc, nn, app)

		if err = update.Apply(db.Ctx, db.Client, &pvc); err != nil {
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

	err = db.Client.Get(db.Ctx, inn, &secret)

	if err != nil {
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
