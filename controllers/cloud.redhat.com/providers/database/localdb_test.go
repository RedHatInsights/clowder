package database

import (
	"fmt"
	"testing"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
)

func getBaseElements() (types.NamespacedName, crd.ClowdApp) {
	nn := types.NamespacedName{
		Name:      "reqapp",
		Namespace: "default",
	}

	objMeta := metav1.ObjectMeta{
		Name:      "reqapp",
		Namespace: "default",
		Labels: p.Labels{
			"app": "test",
		},
	}

	app := crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
				"snapper",
			},
			Deployments: []crd.Deployment{{
				Name: "reqapp",
			}},
		},
	}
	return nn, app
}

func TestLocalDBPVC(t *testing.T) {

	nn, app := getBaseElements()

	pvc := core.PersistentVolumeClaim{}
	provutils.MakeLocalDBPVC(&pvc, nn, &app)

	if pvc.Name != nn.Name {
		t.Fatalf("Name %v did not match expected %v", pvc.Name, nn.Name)
	}
	if pvc.GetLabels()["service"] != "db" {
		t.Fatal("db label was not set")
	}
	accessModeFlag := false
	for _, accessMode := range pvc.Spec.AccessModes {
		if accessMode == core.ReadWriteOnce {
			accessModeFlag = true
		}
	}
	if accessModeFlag != true {
		t.Fatal("Access mode does not equal ReadWriteOnce")
	}
}

func TestLocalDBService(t *testing.T) {
	nn, app := getBaseElements()

	servicePorts := []core.ServicePort{{
		Name:     "database",
		Port:     5432,
		Protocol: "TCP",
	}}

	s := core.Service{}

	labels := &map[string]string{"sub": "test_db"}
	provutils.MakeLocalDBService(&s, nn, &app, labels)

	if s.Name != nn.Name {
		t.Fatalf("Name %v did not match expected %v", s.Name, nn.Name)
	}

	if s.Spec.Ports[0] != servicePorts[0] {
		t.Fatalf("Port did not match the expected database port")
	}
	if s.Spec.Selector["service"] != "db" {
		t.Fatal("db selector was not set")
	}
	if s.Spec.Selector["app"] != app.Name {
		t.Fatal("db app name selector was not set")
	}
}

func TestLocalDBDeployment(t *testing.T) {
	nn, app := getBaseElements()

	cfg := config.DatabaseConfig{
		Hostname:      fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
		Port:          5432,
		Username:      utils.RandString(16),
		Password:      utils.RandString(16),
		AdminPassword: utils.RandString(16),
		Name:          app.Spec.Database.Name,
	}

	envVars := []core.EnvVar{
		{Name: "POSTGRESQL_USER", Value: cfg.Username},
		{Name: "POSTGRESQL_PASSWORD", Value: cfg.Password},
		{Name: "PGPASSWORD", Value: cfg.AdminPassword},
		{Name: "POSTGRESQL_MASTER_USER", Value: cfg.AdminUsername},
		{Name: "POSTGRESQL_MASTER_PASSWORD", Value: cfg.AdminPassword},
		{Name: "POSTGRESQL_DATABASE", Value: app.Spec.Database.Name},
	}

	d := apps.Deployment{}

	image := "imagename:tag"

	labels := &map[string]string{"sub": "test_db"}
	provutils.MakeLocalDB(&d, nn, &app, labels, &cfg, image, true, "")

	if d.Spec.Template.Spec.Containers[0].Image != image {
		t.Fatalf("Image requested %v does not match the one in spec: %v ", image, d.Spec.Template.Spec.Containers[0].Image)
	}
	if d.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != 5432 {
		t.Fatalf("Port requested %v does not match the one in spec: %v ", 5432, d.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	}
	if !compareEnvs(&envVars, &d.Spec.Template.Spec.Containers[0].Env) {
		t.Fatal("Envvars didn't match")
	}
}

func compareEnvs(a, b *([]core.EnvVar)) bool {
	if a == nil && b == nil {
		return true
	} else if len(*a) != len(*b) {
		return false
	}

	envs := make(map[string]string)
	for _, env := range *a {
		envs[env.Name] = env.Value
	}
	for _, env := range *b {
		if envs[env.Name] != env.Value {
			return false
		}
	}
	return true
}
