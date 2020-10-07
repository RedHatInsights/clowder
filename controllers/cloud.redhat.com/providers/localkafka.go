package providers

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type envVar struct {
	Name  string
	Value string
}

type localKafka struct {
	Provider
	Config config.KafkaConfig
}

func (k *localKafka) Configure(config *config.AppConfig) {
	config.Kafka = &k.Config
}

func (k *localKafka) CreateTopic(nn types.NamespacedName, topic *strimzi.KafkaTopicSpec) error {
	topicName := fmt.Sprintf(
		"%s-%s-%s", topic.TopicName, k.Env.Name, k.Env.Spec.Namespace,
	)

	k.Config.Topics = append(
		k.Config.Topics,
		config.TopicConfig{
			Name:          topicName,
			RequestedName: topic.TopicName,
		},
	)

	return nil
}

func NewLocalKafka(p *Provider) (KafkaProvider, error) {
	port := 29092
	config := config.KafkaConfig{
		Topics: []config.TopicConfig{},
		Brokers: []config.BrokerConfig{{
			Hostname: fmt.Sprintf("%v-kafka.%v.svc", p.Env.Name, p.Env.Spec.Namespace),
			Port:     &port,
		}},
	}

	kafkaProvider := localKafka{
		Provider: *p,
		Config:   config,
	}

	if err := makeComponent(p, "zookeeper", makeLocalZookeeper); err != nil {
		return &kafkaProvider, err
	}

	if err := makeComponent(p, "kafka", makeLocalKafka); err != nil {
		return &kafkaProvider, err
	}

	return &kafkaProvider, nil
}

func makeEnvVars(list *[]envVar) []core.EnvVar {

	envVars := []core.EnvVar{}

	for _, ev := range *list {
		envVars = append(envVars, core.EnvVar{Name: ev.Name, Value: ev.Value})
	}

	return envVars
}

func makeLocalKafka(env *crd.ClowdEnvironment, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim) {
	nn := getNamespacedName(env, "kafka")

	labels := env.GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, env)

	labeler(dd)

	dd.Spec.Replicas = utils.Int32(1)
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: nn.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: nn.Name,
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

	envVars := makeEnvVars(&[]envVar{
		{"KAFKA_ADVERTISED_LISTENERS", "PLAINTEXT://" + nn.Name + ":29092, LOCAL://localhost:9092"},
		{"KAFKA_BROKER_ID", "1"},
		{"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR", "1"},
		{"KAFKA_ZOOKEEPER_CONNECT", env.Name + "-zookeeper:32181"},
		{"LOG_DIR", "/var/lib/kafka"},
		{"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP", "PLAINTEXT:PLAINTEXT, LOCAL:PLAINTEXT"},
		{"KAFKA_INTER_BROKER_LISTENER_NAME", "LOCAL"},
	})

	ports := []core.ContainerPort{
		{
			Name:          "kafka",
			ContainerPort: 9092,
		},
	}

	// TODO Readiness and Liveness probes

	c := core.Container{
		Name:  nn.Name,
		Image: "confluentinc/cp-kafka:latest", // TODO: Pull image from quay
		Env:   envVars,
		Ports: ports,
		VolumeMounts: []core.VolumeMount{
			{
				Name:      nn.Name,
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

	servicePorts := []core.ServicePort{{Name: "kafka", Port: 29092, Protocol: "TCP"}}

	utils.MakeService(svc, nn, labels, servicePorts, env)
	utils.MakePVC(pvc, nn, labels, "1Gi", env)
}

func makeLocalZookeeper(env *crd.ClowdEnvironment, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim) {

	nn := getNamespacedName(env, "zookeeper")

	labels := env.GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, env)

	labeler(dd)

	dd.Spec.Replicas = utils.Int32(1)
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: nn.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: nn.Name,
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

	envVars := makeEnvVars(&[]envVar{
		{"ZOOKEEPER_INIT_LIMIT", "10"},
		{"ZOOKEEPER_CLIENT_PORT", "32181"},
		{"ZOOKEEPER_SERVER_ID", "1"},
		{"ZOOKEEPER_SERVERS", nn.Name + ":32181"},
		{"ZOOKEEPER_TICK_TIME", "2000"},
		{"ZOOKEEPER_SYNC_LIMIT", "10"},
	})

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
		Name:  nn.Name,
		Image: "confluentinc/cp-zookeeper:5.3.2",
		Env:   envVars,
		Ports: ports,
		VolumeMounts: []core.VolumeMount{
			{
				Name:      nn.Name,
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

	utils.MakeService(svc, nn, labels, servicePorts, env)
	utils.MakePVC(pvc, nn, labels, "1Gi", env)
}
