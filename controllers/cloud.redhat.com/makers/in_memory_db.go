package makers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
)

//InMemoryDBMaker makes the InMemoryDBConfig object
type InMemoryDBMaker struct {
	*Maker
	config config.InMemoryDB
}

//Make function for the InMemoryDBMaker
func (idb *InMemoryDBMaker) Make() (ctrl.Result, error) {
	idb.config = config.InMemoryDB{}

	var f func() (ctrl.Result, error)

	switch idb.Env.Spec.InMemoryDB.Provider {
	case "redis":
		f = idb.redis
	case "app-interface":
		f = idb.appInterface
	}

	if f != nil {
		return f()
	}

	return ctrl.Result{}, nil
}

//ApplyConfig for the InMemoryDBMaker
func (idb *InMemoryDBMaker) ApplyConfig(c *config.AppConfig) {
	c.InMemoryDb = &idb.config
}

func makeRedisDeployment(dd *apps.Deployment, nn types.NamespacedName, pp *crd.ClowdApp) {
	oneReplica := int32(1)

	pp.SetObjectMeta(
		dd,
		crd.Name(nn.Name),
		crd.Namespace(nn.Namespace),
	)
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: pp.GetLabels()}
	dd.Spec.Template.ObjectMeta.Labels = pp.GetLabels()
	dd.Spec.Replicas = &oneReplica

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: "redis:6",
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "redis",
			ContainerPort: 6379,
		}},
	}}
}

func makeRedisService(s *core.Service, nn types.NamespacedName, pp *crd.ClowdApp) {
	servicePorts := []core.ServicePort{{
		Name:     "database",
		Port:     5432,
		Protocol: "TCP",
	}}

	pp.SetObjectMeta(
		s,
		crd.Name(nn.Name),
		crd.Namespace(nn.Namespace),
	)

	s.Spec.Selector = pp.GetLabels()
	s.Spec.Ports = servicePorts
}

func (idb *InMemoryDBMaker) redis() (ctrl.Result, error) {

	result := ctrl.Result{}

	if !idb.App.Spec.InMemoryDB {
		return result, nil
	}

	nn := idb.App.GetNamespacedName("%v-redis")

	dd := apps.Deployment{}

	update, err := idb.Get(nn, &dd)
	if err != nil {
		return result, err
	}

	makeRedisDeployment(&dd, nn, idb.App)

	if result, err = update.Apply(&dd); err != nil {
		return result, err
	}

	s := core.Service{}
	update, err = idb.Get(nn, &s)
	if err != nil {
		return result, err
	}

	makeRedisService(&s, nn, idb.App)

	if result, err = update.Apply(&s); err != nil {
		return result, err
	}

	return result, nil
}

func (idb *InMemoryDBMaker) appInterface() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
