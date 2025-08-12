package database

import (
	"fmt"
	"testing"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	"github.com/stretchr/testify/assert"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	p "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
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
	volCapacity := sizing.GetDefaultVolCapacity()
	provutils.MakeLocalDBPVC(&pvc, nn, &app, volCapacity)

	assert.Equal(t, nn.Name, pvc.Name, "name did not match expected")
	assert.Equal(t, "db", pvc.GetLabels()["service"], "db label was not set")

	accessModeFlag := false
	for _, accessMode := range pvc.Spec.AccessModes {
		if accessMode == core.ReadWriteOnce {
			accessModeFlag = true
		}
	}

	assert.True(t, accessModeFlag, "access mode does not equal ReadWriteOnce")
}

func TestLocalDBService(t *testing.T) {
	nn, app := getBaseElements()

	servicePorts := []core.ServicePort{{
		Name:       "database",
		Port:       5432,
		Protocol:   core.ProtocolTCP,
		TargetPort: intstr.FromInt(5432),
	}}

	s := core.Service{}

	labels := &map[string]string{"sub": "test_db"}
	provutils.MakeLocalDBService(&s, nn, &app, labels)

	assert.Equal(t, s.Name, nn.Name, "name did not match expected")
	assert.Equal(t, servicePorts[0], s.Spec.Ports[0], "port did not match the expected database port")
	assert.Equal(t, "db", s.Spec.Selector["service"], "db selector was not set")
	assert.Equal(t, app.Name, s.Spec.Selector["app"], "db app name selector was not set")
}

func TestLocalDBDeployment(t *testing.T) {
	nn, app := getBaseElements()

	password, err := utils.RandPassword(16)
	assert.NoError(t, err, "password generate failed")

	adminPassword, err := utils.RandPassword(16)
	assert.NoError(t, err, "adminPassword generate failed")

	cfg := config.DatabaseConfig{
		Hostname:      fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
		Port:          5432,
		Username:      utils.RandString(16),
		Password:      password,
		AdminPassword: adminPassword,
		Name:          app.Spec.Database.Name,
	}

	envVars := []core.EnvVar{
		{Name: "POSTGRESQL_USER", Value: cfg.Username},
		{Name: "POSTGRESQL_PASSWORD", Value: cfg.Password},
		{Name: "POSTGRESQL_ADMIN_PASSWORD", Value: cfg.AdminPassword},
		{Name: "POSTGRESQL_DATABASE", Value: app.Spec.Database.Name},
	}

	d := apps.Deployment{}

	image := "imagename:tag"

	labels := &map[string]string{"sub": "test_db"}
	provutils.MakeLocalDB(&d, nn, &app, labels, &cfg, image, true, "", nil)

	assert.Equal(t, image, d.Spec.Template.Spec.Containers[0].Image, "image requested does not match the one in spec")
	assert.Equal(t, int32(5432), d.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort, "port requested does not match the one in spec")
	assert.Equal(t, &d.Spec.Template.Spec.Containers[0].Env, &envVars, "envvars didn't match")
}
