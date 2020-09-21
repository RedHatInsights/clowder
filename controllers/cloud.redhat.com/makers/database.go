package makers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	ctrl "sigs.k8s.io/controller-runtime"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//DatabaseMaker makes the DatabaseConfig object
type DatabaseMaker struct {
	*Maker
	config config.DatabaseConfig
}

//Make function for the DatabaseMaker
func (db *DatabaseMaker) Make() (ctrl.Result, error) {
	db.config = config.DatabaseConfig{}

	var f func() (ctrl.Result, error)

	switch db.Base.Spec.Database.Provider {
	case "app-interface":
		f = db.appInterface
	case "local":
		f = db.local
	}

	if f != nil {
		return f()
	}

	return ctrl.Result{}, nil

}

//ApplyConfig for the DatabaseMaker
func (db *DatabaseMaker) ApplyConfig(c *config.AppConfig) {
	c.Database = &db.config
}

func (db *DatabaseMaker) appInterface() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func makeLocalDB(dd *apps.Deployment, nn types.NamespacedName, pp *crd.InsightsApp, cfg *config.DatabaseConfig, image string) {
	labels := pp.GetLabels()
	labels["service"] = "db"

	pp.SetObjectMeta(dd, crd.Name(nn.Name), crd.Labels(labels))
	dd.Spec.Replicas = pp.Spec.MinReplicas
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
		{Name: "POSTGRESQL_DATABASE", Value: pp.Spec.Database.Name},
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

func makeLocalService(s *core.Service, nn types.NamespacedName, pp *crd.InsightsApp) {
	servicePorts := []core.ServicePort{{
		Name:     "database",
		Port:     5432,
		Protocol: "TCP",
	}}

	labels := pp.GetLabels()
	labels["service"] = "db"
	pp.SetObjectMeta(s, crd.Name(nn.Name), crd.Namespace(nn.Namespace), crd.Labels(labels))
	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts
}

func makeLocalPVC(pvc *core.PersistentVolumeClaim, nn types.NamespacedName, pp *crd.InsightsApp) {
	labels := pp.GetLabels()
	labels["service"] = "db"
	pp.SetObjectMeta(pvc, crd.Name(nn.Name), crd.Labels(labels))
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}
}

func (db *DatabaseMaker) local() (ctrl.Result, error) {
	result := ctrl.Result{}

	if db.App.Spec.Database == (crd.InsightsDatabaseSpec{}) {
		return result, nil
	}

	nn := db.App.GetNamespacedName("%v-db")

	dd := apps.Deployment{}
	update, err := db.Get(nn, &dd)
	if err != nil {
		return result, err
	}

	db.config = *config.NewDatabaseConfig(db.App.Spec.Database.Name, nn.Name)

	if update.Updater() {
		cfg, err := db.getConfig()
		if err != nil {
			return result, err
		}
		db.config.Username = cfg.Database.Username
		db.config.Password = cfg.Database.Password
		db.config.PgPass = cfg.Database.PgPass
	}

	makeLocalDB(&dd, nn, db.App, &db.config, db.Base.Spec.Database.Image)

	if result, err = update.Apply(&dd); err != nil {
		return result, err
	}

	s := core.Service{}
	update, err = db.Get(nn, &s)
	if err != nil {
		return result, err
	}

	makeLocalService(&s, nn, db.App)

	if result, err = update.Apply(&s); err != nil {
		return result, err
	}

	pvc := core.PersistentVolumeClaim{}
	update, err = db.Get(nn, &pvc)
	if err != nil {
		return result, err
	}

	makeLocalPVC(&pvc, nn, db.App)

	if result, err = update.Apply(&pvc); err != nil {
		return result, err
	}

	return result, nil
}
