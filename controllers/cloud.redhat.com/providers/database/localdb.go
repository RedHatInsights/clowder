package database

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type localDbProvider struct {
	p.Provider
	Config *config.DatabaseConfig
}

func (db *localDbProvider) Configure(c *config.AppConfig) {
	c.Database = db.Config
}

func NewLocalDBProvider(p *p.Provider) (DatabaseProvider, error) {
	return &localDbProvider{Provider: *p}, nil
}

// CreateDatabase ensures a database is created for the given app.  The
// namespaced name passed in must be the actual name of the db resources
func (db *localDbProvider) CreateDatabase(app *crd.ClowdApp) error {
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

	secMap, err := config.MakeOrGetSecret(db.Ctx, db.Env, db.Client, nn, dataInit)
	if err != nil {
		return errors.Wrap("Couldn't set/get secret", err)
	}
	dbCfg.Populate(secMap)

	db.Config = &dbCfg

	makeLocalDB(&dd, nn, app, &dbCfg, db.Env.Spec.Database.Image)

	if err = exists.Apply(db.Ctx, db.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	update, err := utils.UpdateOrErr(db.Client.Get(db.Ctx, nn, &s))

	if err != nil {
		return err
	}

	makeLocalService(&s, nn, app)

	if err = update.Apply(db.Ctx, db.Client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}
	update, err = utils.UpdateOrErr(db.Client.Get(db.Ctx, nn, &pvc))

	if err != nil {
		return err
	}

	makeLocalPVC(&pvc, nn, app)

	if err = update.Apply(db.Ctx, db.Client, &pvc); err != nil {
		return err
	}

	return nil
}

func makeLocalDB(dd *apps.Deployment, nn types.NamespacedName, app *crd.ClowdApp, cfg *config.DatabaseConfig, image string) {
	labels := app.GetLabels()
	labels["service"] = "db"
	labler := utils.MakeLabeler(nn, labels, app)
	labler(dd)
	dd.Spec.Replicas = utils.Int32(1)
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: nn.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: nn.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = labels

	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{{
		Name: "quay-cloudservices-pull",
	}}

	envVars := []core.EnvVar{
		{Name: "POSTGRESQL_USER", Value: cfg.Username},
		{Name: "POSTGRESQL_PASSWORD", Value: cfg.Password},
		{Name: "PGPASSWORD", Value: cfg.PgPass},
		{Name: "POSTGRESQL_DATABASE", Value: app.Spec.Database.Name},
	}
	ports := []core.ContainerPort{{
		Name:          "database",
		ContainerPort: 5432,
	}}

	probeHandler := core.Handler{
		Exec: &core.ExecAction{
			Command: []string{
				"psql",
				"-U",
				"$(POSTGRESQL_USER)",
				"-d",
				"$(POSTGRESQL_DATABASE)",
				"-c",
				"SELECT 1",
			},
		},
	}

	livenessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 15,
		TimeoutSeconds:      2,
	}
	readinessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 45,
		TimeoutSeconds:      2,
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          image,
		Env:            envVars,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		Ports:          ports,
		VolumeMounts: []core.VolumeMount{{
			Name:      nn.Name,
			MountPath: "/var/lib/pgsql/data",
		}},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
}

func makeLocalService(s *core.Service, nn types.NamespacedName, app *crd.ClowdApp) {
	servicePorts := []core.ServicePort{{
		Name:     "database",
		Port:     5432,
		Protocol: "TCP",
	}}
	utils.MakeService(s, nn, p.Labels{"service": "db"}, servicePorts, app)
}

func makeLocalPVC(pvc *core.PersistentVolumeClaim, nn types.NamespacedName, app *crd.ClowdApp) {
	utils.MakePVC(pvc, nn, p.Labels{"service": "db"}, "1Gi", app)
}
