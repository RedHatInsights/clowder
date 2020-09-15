package makers

import (
	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"
	"fmt"
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

	redisObjName := fmt.Sprintf("%v-redis", idb.App.Name)
	redisNamespacedName := types.NamespacedName{
		Namespace: idb.App.Namespace,
		Name:      redisObjName,
	}

	dd := apps.Deployment{}
	err := idb.Client.Get(idb.Ctx, redisNamespacedName, &dd)

	update, err := utils.UpdateOrErr(err)

	if err != nil {
		return err
	}

	oneReplica := int32(1)

	dd.SetName(redisNamespacedName.Name)
	dd.SetNamespace(redisNamespacedName.Namespace)
	dd.SetLabels(idb.App.GetLabels())
	dd.SetOwnerReferences([]metav1.OwnerReference{idb.App.MakeOwnerReference()})
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: idb.App.GetLabels()}
	dd.Spec.Template.ObjectMeta.Labels = idb.App.GetLabels()
	dd.Spec.Replicas = &oneReplica

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  redisNamespacedName.Name,
		Image: "redis:6",
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "redis",
			ContainerPort: 6379,
		}},
	}}

	if err = update.Apply(idb.Ctx, idb.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	err = idb.Client.Get(idb.Ctx, redisNamespacedName, &s)

	update, err = utils.UpdateOrErr(err)
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
		crd.Name(redisNamespacedName.Name),
		crd.Namespace(redisNamespacedName.Namespace),
	)

	s.Spec.Selector = idb.App.GetLabels()
	s.Spec.Ports = servicePorts

	if err = update.Apply(idb.Ctx, idb.Client, &s); err != nil {
		return err
	}

	return nil
}

func (idb *InMemoryDBMaker) appInterface() error {
	return nil
}
