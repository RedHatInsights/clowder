/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"
)

// InsightsBaseReconciler reconciles a InsightsBase object
type InsightsBaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsbases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsbases/status,verbs=get;update;patch

func (r *InsightsBaseReconciler) makeMinio(ctx context.Context, req ctrl.Request, base *crd.InsightsBase) error {
	minioObjName := fmt.Sprintf("%v-minio", req.Name)
	minioNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      minioObjName,
	}

	dd := apps.Deployment{}
	err := r.Client.Get(ctx, minioNamespacedName, &dd)
	update, err := utils.UpdateOrErr(err)

	if err != nil {
		return err
	}

	labels := base.GetLabels()
	labels["base-app"] = minioNamespacedName.Name

	dd.SetName(minioNamespacedName.Name)
	dd.SetNamespace(minioNamespacedName.Namespace)
	dd.SetLabels(labels)
	dd.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: minioNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: minioNamespacedName.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = labels

	pullSecretRef := core.LocalObjectReference{Name: "quay-cloudservices-pull"}
	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{pullSecretRef}

	var accessKey, secretKey core.EnvVar
	if !update {
		accessKey = core.EnvVar{Name: "MINIO_ACCESS_KEY", Value: utils.RandString(12)}
		secretKey = core.EnvVar{Name: "MINIO_SECRET_KEY", Value: utils.RandString(12)}
		annotations := base.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["accessKey"] = accessKey.Value
		annotations["secretKey"] = secretKey.Value
		annotations["endpoint"] = fmt.Sprintf("%v.%v.svc:9000", minioObjName, req.Namespace)
		base.SetAnnotations(annotations)
		if err = r.Client.Update(ctx, base); err != nil {
			return err
		}
	} else {
		appConfig := base.GetAnnotations()
		if err != nil {
			return err
		}
		accessKey = core.EnvVar{Name: "MINIO_ACCESS_KEY", Value: appConfig["accessKey"]}
		secretKey = core.EnvVar{Name: "MINIO_SECRET_KEY", Value: appConfig["secretKey"]}
	}
	envVars := []core.EnvVar{accessKey, secretKey}
	ports := []core.ContainerPort{
		{
			Name:          "minio",
			ContainerPort: 9000,
		},
	}

	// TODO Readiness and Liveness probes

	c := core.Container{
		Name:  minioNamespacedName.Name,
		Image: "minio/minio",
		Env:   envVars,
		Ports: ports,
		VolumeMounts: []core.VolumeMount{{
			Name:      minioNamespacedName.Name,
			MountPath: "/storage",
		}},
		Args: []string{
			"server",
			"/storage",
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	if err = update.Apply(ctx, r.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	err = r.Client.Get(ctx, minioNamespacedName, &s)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{}
	minioPort := core.ServicePort{Name: "minio", Port: 9000, Protocol: "TCP"}
	servicePorts = append(servicePorts, minioPort)

	s.SetName(minioNamespacedName.Name)
	s.SetNamespace(minioNamespacedName.Namespace)
	s.SetLabels(labels)
	s.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts

	if err = update.Apply(ctx, r.Client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}

	err = r.Client.Get(ctx, minioNamespacedName, &pvc)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	pvc.SetName(minioNamespacedName.Name)
	pvc.SetNamespace(minioNamespacedName.Namespace)
	pvc.SetLabels(labels)
	pvc.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(ctx, r.Client, &pvc); err != nil {
		return err
	}
	return nil
}

func (r *InsightsBaseReconciler) makeKafka(ctx context.Context, req ctrl.Request, base *crd.InsightsBase) error {
	kafkaObjName := fmt.Sprintf("%v-kafka", req.Name)
	kafkaNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      kafkaObjName,
	}

	dd := apps.Deployment{}
	err := r.Client.Get(ctx, kafkaNamespacedName, &dd)
	update, err := utils.UpdateOrErr(err)

	if err != nil {
		return err
	}

	labels := base.GetLabels()
	labels["base-app"] = kafkaNamespacedName.Name

	dd.SetName(kafkaNamespacedName.Name)
	dd.SetNamespace(kafkaNamespacedName.Namespace)
	dd.SetLabels(labels)
	dd.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: kafkaNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: kafkaNamespacedName.Name,
			},
		}},
		{
			Name: "mq-kafka-1",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "mq-kafka-2",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
	}
	dd.Spec.Template.ObjectMeta.Labels = labels

	envVars := []core.EnvVar{
		{
			Name: "KAFKA_ADVERTISED_LISTENERS", Value: "PLAINTEXT://" + kafkaNamespacedName.Name + ":29092, LOCAL://localhost:9092",
		},
		{
			Name:  "KAFKA_BROKER_ID",
			Value: "1",
		},
		{
			Name:  "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR",
			Value: "1",
		},
		{
			Name:  "KAFKA_ZOOKEEPER_CONNECT",
			Value: base.Name + "-zookeeper:32181",
		},
		{
			Name:  "LOG_DIR",
			Value: "/var/lib/mq-kafka",
		},
		{
			Name:  "KAFKA_LISTENER_SECURITY_PROTOCOL_MAP",
			Value: "PLAINTEXT:PLAINTEXT, LOCAL:PLAINTEXT",
		},
		{
			Name:  "KAFKA_INTER_BROKER_LISTENER_NAME",
			Value: "LOCAL",
		},
	}
	ports := []core.ContainerPort{
		{
			Name:          "kafka",
			ContainerPort: 9092,
		},
	}

	// TODO Readiness and Liveness probes

	c := core.Container{
		Name:  kafkaNamespacedName.Name,
		Image: "confluentinc/cp-kafka:latest",
		Env:   envVars,
		Ports: ports,
		VolumeMounts: []core.VolumeMount{
			{
				Name:      kafkaNamespacedName.Name,
				MountPath: "/var/lib/kafka",
			},
			{
				Name:      "mq-kafka-1",
				MountPath: "/etc/kafka/secrets",
			},
			{
				Name:      "mq-kafka-2",
				MountPath: "/var/lib/kafka/data",
			},
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	if err = update.Apply(ctx, r.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	err = r.Client.Get(ctx, kafkaNamespacedName, &s)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{}
	kafkaPort := core.ServicePort{Name: "kafka", Port: 29092, Protocol: "TCP"}
	servicePorts = append(servicePorts, kafkaPort)

	s.SetName(kafkaNamespacedName.Name)
	s.SetNamespace(kafkaNamespacedName.Namespace)
	s.SetLabels(labels)
	s.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts

	if err = update.Apply(ctx, r.Client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}

	err = r.Client.Get(ctx, kafkaNamespacedName, &pvc)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	pvc.SetName(kafkaNamespacedName.Name)
	pvc.SetNamespace(kafkaNamespacedName.Namespace)
	pvc.SetLabels(labels)
	pvc.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(ctx, r.Client, &pvc); err != nil {
		return err
	}
	return nil
}

func (r *InsightsBaseReconciler) makeZookeeper(ctx context.Context, req ctrl.Request, base *crd.InsightsBase) error {
	zookeeperObjName := fmt.Sprintf("%v-zookeeper", req.Name)
	zookeeperNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      zookeeperObjName,
	}

	dd := apps.Deployment{}
	err := r.Client.Get(ctx, zookeeperNamespacedName, &dd)
	update, err := utils.UpdateOrErr(err)

	if err != nil {
		return err
	}

	labels := base.GetLabels()
	labels["base-app"] = zookeeperNamespacedName.Name

	dd.SetName(zookeeperNamespacedName.Name)
	dd.SetNamespace(zookeeperNamespacedName.Namespace)
	dd.SetLabels(labels)
	dd.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: zookeeperNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: zookeeperNamespacedName.Name,
			},
		}},
		{
			Name: "mq-zookeeper-1",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "mq-zookeeper-2",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "mq-zookeeper-3",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
	}
	dd.Spec.Template.ObjectMeta.Labels = labels

	envVars := []core.EnvVar{
		{
			Name:  "ZOOKEEPER_INIT_LIMIT",
			Value: "10",
		},
		{
			Name:  "ZOOKEEPER_CLIENT_PORT",
			Value: "32181",
		},
		{
			Name:  "ZOOKEEPER_SERVER_ID",
			Value: "1",
		},
		{
			Name:  "ZOOKEEPER_SERVERS",
			Value: zookeeperNamespacedName.Name + ":32181",
		},
		{
			Name:  "ZOOKEEPER_TICK_TIME",
			Value: "2000",
		},
		{
			Name:  "ZOOKEEPER_SYNC_LIMIT",
			Value: "10",
		},
	}
	ports := []core.ContainerPort{
		{
			Name:          "zookeeper",
			ContainerPort: 2181,
		},
		{
			Name:          "zookeeper-1",
			ContainerPort: 2888,
		},
		{
			Name:          "zookeeper-2",
			ContainerPort: 3888,
		},
	}

	// TODO Readiness and Liveness probes

	c := core.Container{
		Name:  zookeeperNamespacedName.Name,
		Image: "confluentinc/cp-zookeeper:5.3.2",
		Env:   envVars,
		Ports: ports,
		VolumeMounts: []core.VolumeMount{
			{
				Name:      zookeeperNamespacedName.Name,
				MountPath: "/var/lib/zookeeper",
			},
			{
				Name:      "mq-zookeeper-1",
				MountPath: "/etc/zookeeper/secrets",
			},
			{
				Name:      "mq-zookeeper-2",
				MountPath: "/var/lib/zookeeper/data",
			},
			{
				Name:      "mq-zookeeper-3",
				MountPath: "/var/lib/zookeeper/log",
			},
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	if err = update.Apply(ctx, r.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	err = r.Client.Get(ctx, zookeeperNamespacedName, &s)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{
		{
			Name: "zookeeper1", Port: 32181, Protocol: "TCP",
		},
		{
			Name: "zookeeper2", Port: 2888, Protocol: "TCP",
		},
		{
			Name: "zookeeper3", Port: 3888, Protocol: "TCP",
		},
	}

	s.SetName(zookeeperNamespacedName.Name)
	s.SetNamespace(zookeeperNamespacedName.Namespace)
	s.SetLabels(labels)
	s.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts

	if err = update.Apply(ctx, r.Client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}

	err = r.Client.Get(ctx, zookeeperNamespacedName, &pvc)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	pvc.SetName(zookeeperNamespacedName.Name)
	pvc.SetNamespace(zookeeperNamespacedName.Namespace)
	pvc.SetLabels(labels)
	pvc.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(ctx, r.Client, &pvc); err != nil {
		return err
	}
	return nil
}

//Reconcile fn
func (r *InsightsBaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("insightsbase", req.NamespacedName)

	base := crd.InsightsBase{}
	err := r.Client.Get(ctx, req.NamespacedName, &base)

	if err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if base.Spec.ObjectStore.Provider == "minio" {
		err = r.makeMinio(ctx, req, &base)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if base.Spec.Kafka.Provider == "local" {
		err = r.makeZookeeper(ctx, req, &base)

		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.makeKafka(ctx, req, &base)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up with manager
func (r *InsightsBaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.InsightsBase{}).
		Complete(r)
}
