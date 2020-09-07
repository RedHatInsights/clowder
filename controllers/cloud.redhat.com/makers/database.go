package makers

import (
	"fmt"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"

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
func (db *DatabaseMaker) Make() error {
	db.config = config.DatabaseConfig{}

	var f func() error

	switch db.Base.Spec.Database.Provider {
	case "app-interface":
		f = db.appInterface
	case "local":
		f = db.local
	}

	if f != nil {
		return f()
	}

	return nil

}

//ApplyConfig for the DatabaseMaker
func (db *DatabaseMaker) ApplyConfig(c *config.AppConfig) {
	c.Database = db.config
}

func (db *DatabaseMaker) appInterface() error {
	return nil
}

func (db *DatabaseMaker) local() error {
	if db.App.Spec.Database == (crd.InsightsDatabaseSpec{}) {
		return nil
	}

	dbObjName := fmt.Sprintf("%v-db", db.App.Name)
	dbNamespacedName := types.NamespacedName{
		Namespace: db.App.Namespace,
		Name:      dbObjName,
	}

	dd := apps.Deployment{}
	err := db.Client.Get(db.Ctx, dbNamespacedName, &dd)

	update, err := updateOrErr(err)

	if err != nil {
		return err
	}

	dd.SetName(dbNamespacedName.Name)
	dd.SetNamespace(dbNamespacedName.Namespace)
	dd.SetLabels(db.App.GetLabels())
	dd.SetOwnerReferences([]metav1.OwnerReference{db.App.MakeOwnerReference()})

	dd.Spec.Replicas = db.App.Spec.MinReplicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: db.App.GetLabels()}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: dbNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: dbNamespacedName.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = db.App.GetLabels()

	pullSecretRef := core.LocalObjectReference{Name: "quay-cloudservices-pull"}
	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{pullSecretRef}

	dbUser := core.EnvVar{Name: "POSTGRESQL_USER", Value: "test"}
	dbPass := core.EnvVar{Name: "POSTGRESQL_PASSWORD", Value: "test"}
	dbName := core.EnvVar{Name: "POSTGRESQL_DATABASE", Value: db.App.Spec.Database.Name}
	pgPass := core.EnvVar{Name: "PGPASSWORD", Value: "test"}
	envVars := []core.EnvVar{dbUser, dbPass, dbName, pgPass}
	ports := []core.ContainerPort{
		{
			Name:          "database",
			ContainerPort: 5432,
		},
	}

	livenessProbe := core.Probe{
		Handler: core.Handler{
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
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      2,
	}
	readinessProbe := core.Probe{
		Handler: core.Handler{
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
		},
		InitialDelaySeconds: 45,
		TimeoutSeconds:      2,
	}

	c := core.Container{
		Name:           dbNamespacedName.Name,
		Image:          db.Base.Spec.Database.Image,
		Env:            envVars,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		Ports:          ports,
		VolumeMounts: []core.VolumeMount{{
			Name:      dbNamespacedName.Name,
			MountPath: "/var/lib/pgsql/data",
		}},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}

	if err = update.Apply(db.Ctx, db.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	err = db.Client.Get(db.Ctx, dbNamespacedName, &s)

	update, err = updateOrErr(err)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{}
	databasePort := core.ServicePort{Name: "database", Port: 5432, Protocol: "TCP"}
	servicePorts = append(servicePorts, databasePort)

	db.App.SetObjectMeta(&s, crd.Name(dbNamespacedName.Name), crd.Namespace(dbNamespacedName.Namespace))
	s.Spec.Selector = db.App.GetLabels()
	s.Spec.Ports = servicePorts

	if err = update.Apply(db.Ctx, db.Client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}

	err = db.Client.Get(db.Ctx, dbNamespacedName, &pvc)

	update, err = updateOrErr(err)
	if err != nil {
		return err
	}

	pvc.SetName(dbNamespacedName.Name)
	pvc.SetNamespace(dbNamespacedName.Namespace)
	pvc.SetLabels(db.App.GetLabels())
	pvc.SetOwnerReferences([]metav1.OwnerReference{db.App.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(db.Ctx, db.Client, &pvc); err != nil {
		return err
	}

	db.config.Name = db.App.Spec.Database.Name
	db.config.User = dbUser.Value
	db.config.Pass = dbPass.Value
	db.config.Hostname = dbObjName
	db.config.Port = 5432

	return nil
}
