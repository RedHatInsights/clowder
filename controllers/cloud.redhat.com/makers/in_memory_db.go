package makers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	var f func() error

	switch idb.Base.Spec.InMemoryDB.Provider {
	case "redis":
		f = idb.redis
	case "app-interface":
		f = idb.appInterface
	}

	if f != nil {
		return ctrl.Result{}, f()
	}

	return ctrl.Result{}, nil
}

//ApplyConfig for the InMemoryDBMaker
func (idb *InMemoryDBMaker) ApplyConfig(c *config.AppConfig) {
	c.InMemoryDb = &idb.config
}

func (idb *InMemoryDBMaker) redis() error {
	if !idb.App.Spec.InMemoryDB {
		return nil
	}

	nn := idb.GetNamespacedName("%v-redis")

	dd := apps.Deployment{}

	update, err := idb.Get(nn, &dd)
	if err != nil {
		return err
	}

	oneReplica := int32(1)

	dd.SetName(nn.Name)
	dd.SetNamespace(nn.Namespace)
	dd.SetLabels(idb.App.GetLabels())
	dd.SetOwnerReferences([]metav1.OwnerReference{idb.App.MakeOwnerReference()})
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: idb.App.GetLabels()}
	dd.Spec.Template.ObjectMeta.Labels = idb.App.GetLabels()
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

	if err = update.Apply(&dd); err != nil {
		return err
	}

	s := core.Service{}
	update, err = idb.Get(nn, &s)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{{
		Name:     "database",
		Port:     5432,
		Protocol: "TCP",
	}}

	idb.App.SetObjectMeta(
		&s,
		crd.Name(nn.Name),
		crd.Namespace(nn.Namespace),
	)

	s.Spec.Selector = idb.App.GetLabels()
	s.Spec.Ports = servicePorts

	if err = update.Apply(&s); err != nil {
		return err
	}

	return nil
}

func (idb *InMemoryDBMaker) appInterface() error {
	return nil
}
