package controllers

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/whippoorwill/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func b64decode(s *core.Secret, key string, log logr.Logger) string {
	decoded, err := b64.StdEncoding.DecodeString(string(s.Data[key]))

	if err != nil {
		log.Error(err, "Based to base64 decode")
	}

	return string(decoded)
}

type Maker struct {
	App     *crd.InsightsApp
	Base    *crd.InsightsBase
	Client  client.Client
	Ctx     context.Context
	Request *ctrl.Request
}

func (m *Maker) makeKafka() error {
	if len(m.App.Spec.KafkaTopics) > -1 {
		for _, kafkaTopic := range m.App.Spec.KafkaTopics {
			k := strimzi.KafkaTopic{}
			kafkaNamespace := types.NamespacedName{
				Namespace: m.Base.Spec.KafkaNamespace,
				Name:      kafkaTopic.TopicName,
			}

			err := m.Client.Get(m.Ctx, kafkaNamespace, &k)
			update, err := updateOrErr(err)
			if err != nil {
				return err
			}

			labels := map[string]string{
				"strimzi.io/cluster": m.Base.Spec.KafkaCluster,
				"iapp":               m.App.GetName(), // If we label it with the app name, since app names should be unique? can we use for delete selector?
			}

			k.SetName(kafkaTopic.TopicName)
			k.SetNamespace(m.Base.Spec.KafkaNamespace)
			k.SetLabels(labels)

			k.Spec.Replicas = kafkaTopic.Replicas
			k.Spec.Partitions = kafkaTopic.Partitions
			k.Spec.Config = kafkaTopic.Config
			err = update.Apply(m.Ctx, m.Client, &k)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Maker) makeService() error {

	s := core.Service{}
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &s)

	update, err := updateOrErr(err)
	if err != nil {
		return err
	}

	ports := []core.ServicePort{
		{Name: "metrics", Port: m.Base.Spec.MetricsPort, Protocol: "TCP"},
	}

	if m.App.Spec.Web == true {
		webPort := core.ServicePort{Name: "web", Port: m.Base.Spec.WebPort, Protocol: "TCP"}
		ports = append(ports, webPort)
	}

	m.App.SetObjectMeta(&s)
	s.Spec.Selector = m.App.GetLabels()
	s.Spec.Ports = ports

	return update.Apply(m.Ctx, m.Client, &s)
}

func (m *Maker) makeDatabase() (config.DatabaseConfig, error) {
	// TODO Right now just dealing with the creation for ephemeral - doesn't skip if RDS

	dbObjName := fmt.Sprintf("%v-db", m.App.Name)
	dbNamespacedName := types.NamespacedName{
		Namespace: m.App.Namespace,
		Name:      dbObjName,
	}

	dd := apps.Deployment{}
	err := m.Client.Get(m.Ctx, dbNamespacedName, &dd)

	update, err := updateOrErr(err)
	if err != nil {
		return config.DatabaseConfig{}, err
	}

	dd.SetName(dbNamespacedName.Name)
	dd.SetNamespace(dbNamespacedName.Namespace)
	dd.SetLabels(m.App.GetLabels())
	dd.SetOwnerReferences([]metav1.OwnerReference{m.App.MakeOwnerReference()})

	dd.Spec.Replicas = m.App.Spec.MinReplicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: m.App.GetLabels()}
	dd.Spec.Template.Spec.Volumes = []core.Volume{core.Volume{
		Name: dbNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: dbNamespacedName.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = m.App.GetLabels()

	pullSecretRef := core.LocalObjectReference{Name: "quay-cloudservices-pull"}
	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{pullSecretRef}

	dbUser := core.EnvVar{Name: "POSTGRESQL_USER", Value: "test"}
	dbPass := core.EnvVar{Name: "POSTGRESQL_PASSWORD", Value: "test"}
	dbName := core.EnvVar{Name: "POSTGRESQL_DATABASE", Value: m.App.Spec.Database.Name}
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
		Image:          m.Base.Spec.DatabaseImage,
		Env:            envVars,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		// VolumeMounts:   m.App.Spec.VolumeMounts, TODO Add in volume mount for PVC
		Ports: ports,
		VolumeMounts: []core.VolumeMount{
			core.VolumeMount{
				Name:      dbNamespacedName.Name,
				MountPath: "/var/lib/pgsql/data",
			},
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}

	if err = update.Apply(m.Ctx, m.Client, &dd); err != nil {
		return config.DatabaseConfig{}, err
	}

	s := core.Service{}
	err = m.Client.Get(m.Ctx, dbNamespacedName, &s)

	update, err = updateOrErr(err)
	if err != nil {
		return config.DatabaseConfig{}, err
	}

	servicePorts := []core.ServicePort{}
	databasePort := core.ServicePort{Name: "database", Port: 5432, Protocol: "TCP"}
	servicePorts = append(servicePorts, databasePort)

	m.App.SetObjectMeta(&s, crd.Name(dbNamespacedName.Name), crd.Namespace(dbNamespacedName.Namespace))
	s.Spec.Selector = m.App.GetLabels()
	s.Spec.Ports = servicePorts

	if err = update.Apply(m.Ctx, m.Client, &s); err != nil {
		return config.DatabaseConfig{}, err
	}

	pvc := core.PersistentVolumeClaim{}

	err = m.Client.Get(m.Ctx, dbNamespacedName, &pvc)

	update, err = updateOrErr(err)
	if err != nil {
		return config.DatabaseConfig{}, err
	}

	pvc.SetName(dbNamespacedName.Name)
	pvc.SetNamespace(dbNamespacedName.Namespace)
	pvc.SetLabels(m.App.GetLabels())
	pvc.SetOwnerReferences([]metav1.OwnerReference{m.App.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(m.Ctx, m.Client, &pvc); err != nil {
		return config.DatabaseConfig{}, err
	}

	dbConfig := config.DatabaseConfig{
		Name:     m.App.Spec.Database.Name,
		User:     dbUser.Value,
		Pass:     dbPass.Value,
		Hostname: dbObjName,
		Port:     5432,
	}

	return dbConfig, nil
}

func (m *Maker) persistConfig(c *config.AppConfig) error {

	// In any case, we want to overwrite the secret, so this just
	// tests to see if the secret exists
	err := m.Client.Get(m.Ctx, m.Request.NamespacedName, &core.Secret{})

	update, err := updateOrErr(err)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(c)
	if err != nil {
		return err
	}

	secret := core.Secret{
		StringData: map[string]string{
			"cdappconfig.json": string(jsonData),
		},
	}

	m.App.SetObjectMeta(&secret)

	return update.Apply(m.Ctx, m.Client, &secret)
}

func (m *Maker) configureLogging(c *config.AppConfig, log logr.Logger) error {

	if m.Base.Spec.Logging == "cloudwatch" {
		name := types.NamespacedName{
			Name:      "cloudwatch",
			Namespace: m.App.Namespace,
		}

		secret := core.Secret{}
		err := m.Client.Get(m.Ctx, name, &secret)

		if err != nil {
			return err
		}

		c.CloudWatch = config.CloudWatchConfig{
			AccessKeyID:     b64decode(&secret, "aws_access_key_id", log),
			SecretAccessKey: b64decode(&secret, "aws_secret_access_key", log),
			Region:          b64decode(&secret, "aws_region", log),
			LogGroup:        b64decode(&secret, "log_group_name", log),
		}
	}

	return nil
}

// This should probably take arguments for addtional volumes, so that we can add those and then do one Apply
func (m *Maker) makeDeployment(d *apps.Deployment) {

	m.App.SetObjectMeta(d)

	d.Spec.Replicas = m.App.Spec.MinReplicas
	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: m.App.GetLabels()}
	d.Spec.Template.Spec.Volumes = m.App.Spec.Volumes
	d.Spec.Template.ObjectMeta.Labels = m.App.GetLabels()

	d.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
		{Name: "quay-cloudservices-pull"},
	}

	c := core.Container{
		Name:           m.App.ObjectMeta.Name,
		Image:          m.App.Spec.Image,
		Command:        m.App.Spec.Command,
		Args:           m.App.Spec.Args,
		Env:            m.App.Spec.Env,
		Resources:      m.App.Spec.Resources,
		LivenessProbe:  m.App.Spec.LivenessProbe,
		ReadinessProbe: m.App.Spec.ReadinessProbe,
		VolumeMounts:   m.App.Spec.VolumeMounts,
		Ports: []core.ContainerPort{{
			Name:          "metrics",
			ContainerPort: m.Base.Spec.MetricsPort,
		}},
	}

	if m.App.Spec.Web {
		c.Ports = append(c.Ports, core.ContainerPort{
			Name:          "web",
			ContainerPort: m.Base.Spec.WebPort,
		})
	}

	d.Spec.Template.Spec.Containers = []core.Container{c}
}
