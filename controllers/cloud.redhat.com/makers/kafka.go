package makers

import (
	"fmt"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/whippoorwill/v2/apis/kafka.strimzi.io/v1beta1"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var conversionMap = map[string]func([]string) (string, error){
	"retention.ms":          utils.IntMax,
	"retention.bytes":       utils.IntMax,
	"min.compaction.lag.ms": utils.IntMax,
	"cleanup.policy":        utils.ListMerge,
}

//KafkaMaker makes the KafkaConfig object
type KafkaMaker struct {
	*Maker
	config config.KafkaConfig
}

// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kafka.strimzi.io,resources=kafkas,verbs=get;list;watch

//Make function for the KafkaMaker
func (k *KafkaMaker) Make() (ctrl.Result, error) {
	k.config = config.KafkaConfig{}

	var f func() error

	switch k.Base.Spec.Kafka.Provider {
	case "operator":
		f = k.operator
	case "local":
		f = k.local
	}

	if f != nil {
		return ctrl.Result{}, f()
	}

	return ctrl.Result{}, nil
}

//ApplyConfig for the KafkaMaker
func (k *KafkaMaker) ApplyConfig(c *config.AppConfig) {
	c.Kafka = &k.config
}

func (k *KafkaMaker) local() error {

	k.config.Topics = []config.TopicConfig{}
	k.config.Brokers = []config.BrokerConfig{}

	appList := crd.InsightsAppList{}
	err := k.Client.List(k.Ctx, &appList)

	if err != nil {
		return err
	}

	bc := config.BrokerConfig{
		Hostname: k.Base.Name + "-kafka." + k.Request.Namespace + ".svc",
	}
	port := 29092
	p := int(port)
	bc.Port = &p
	k.config.Brokers = append(k.config.Brokers, bc)
	for _, kafkaTopic := range k.App.Spec.KafkaTopics {

		topicName := fmt.Sprintf("%s-%s-%s", kafkaTopic.TopicName, k.Base.Name, k.Request.Namespace)

		k.config.Topics = append(
			k.config.Topics,
			config.TopicConfig{Name: topicName, RequestedName: kafkaTopic.TopicName},
		)
	}
	return nil
}

func (k *KafkaMaker) operator() error {
	if k.App.Spec.KafkaTopics == nil {
		return nil
	}

	k.config.Topics = []config.TopicConfig{}
	k.config.Brokers = []config.BrokerConfig{}

	appList := crd.InsightsAppList{}
	err := k.Client.List(k.Ctx, &appList)

	if err != nil {
		return err
	}

	for _, kafkaTopic := range k.App.Spec.KafkaTopics {
		kRes := strimzi.KafkaTopic{}

		topicName := fmt.Sprintf("%s-%s-%s", kafkaTopic.TopicName, k.Base.Name, k.Request.Namespace)

		update, err := k.Get(types.NamespacedName{
			Namespace: k.Base.Spec.Kafka.Namespace,
			Name:      topicName,
		}, &kRes)

		if err != nil {
			return err
		}

		labels := map[string]string{
			"strimzi.io/cluster": k.Base.Spec.Kafka.ClusterName,
			"iapp":               k.App.GetName(),
			// If we label it with the app name, since app names should be
			// unique? can we use for delete selector?
		}

		kRes.SetName(topicName)
		kRes.SetNamespace(k.Base.Spec.Kafka.Namespace)
		kRes.SetLabels(labels)

		kRes.Spec.Replicas = kafkaTopic.Replicas
		kRes.Spec.Partitions = kafkaTopic.Partitions
		kRes.Spec.Config = kafkaTopic.Config

		newConfig := make(map[string]string)

		// This can be improved from an efficiency PoV
		// Loop through all key/value pairs in the config
		for key, value := range kRes.Spec.Config {
			valList := []string{value}
			for _, res := range appList.Items {
				if res.ObjectMeta.Name == k.Request.Name {
					continue
				}
				if res.ObjectMeta.Namespace != k.Request.Namespace {
					continue
				}
				if res.Spec.KafkaTopics != nil {
					for _, topic := range res.Spec.KafkaTopics {
						if topic.Config != nil {
							if val, ok := topic.Config[key]; ok {
								valList = append(valList, val)
							}
						}
					}
				}
			}
			f, ok := conversionMap[key]
			if ok {
				out, _ := f(valList)
				newConfig[key] = out
			} else {
				err = fmt.Errorf("no conversion type for %s", key)
				return err
			}
		}

		kRes.Spec.Config = newConfig

		if err = update.Apply(&kRes); err != nil {
			return err
		}

		k.config.Topics = append(
			k.config.Topics,
			config.TopicConfig{Name: topicName, RequestedName: kafkaTopic.TopicName},
		)
	}

	clusterName := types.NamespacedName{
		Namespace: k.Base.Spec.Kafka.Namespace,
		Name:      k.Base.Spec.Kafka.ClusterName,
	}

	kafkaResource := strimzi.Kafka{}
	if _, err := k.Get(clusterName, &kafkaResource); err != nil {
		return err
	}

	for _, listener := range kafkaResource.Status.Listeners {
		if listener.Type == "plain" {
			bc := config.BrokerConfig{
				Hostname: listener.Addresses[0].Host,
			}
			port := listener.Addresses[0].Port
			if port != nil {
				p := int(*port)
				bc.Port = &p
			}
			k.config.Brokers = append(k.config.Brokers, bc)
		}
	}

	return nil
}

func MakeLocalKafka(maker *Maker) error {
	nn := GetNamespacedName(maker.Request, "%v-kafka")

	dd := apps.Deployment{}
	update, err := maker.Get(nn, &dd)
	if err != nil {
		return err
	}

	labels := maker.Base.GetLabels()
	labels["base-app"] = nn.Name

	labeler := maker.MakeLabeler(nn, labels)

	labeler(&dd)

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

	envVars := []core.EnvVar{
		{
			Name: "KAFKA_ADVERTISED_LISTENERS", Value: "PLAINTEXT://" + nn.Name + ":29092, LOCAL://localhost:9092",
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
			Value: maker.Base.Name + "-zookeeper:32181",
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
		Name:  nn.Name,
		Image: "confluentinc/cp-kafka:latest",
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

	if err = update.Apply(&dd); err != nil {
		return err
	}

	s := core.Service{}
	update, err = maker.Get(nn, &s)
	if err != nil {
		return err
	}

	labeler(&s)

	s.Spec.Selector = labels
	s.Spec.Ports = []core.ServicePort{{Name: "kafka", Port: 29092, Protocol: "TCP"}}

	if err = update.Apply(&s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}
	update, err = maker.Get(nn, &pvc)
	if err != nil {
		return err
	}

	labeler(&pvc)

	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(&pvc); err != nil {
		return err
	}
	return nil
}

func MakeLocalZookeeper(maker *Maker) error {

	nn := GetNamespacedName(maker.Request, "%v-zookeeper")

	dd := apps.Deployment{}
	update, err := maker.Get(nn, &dd)

	if err != nil {
		return err
	}

	labels := maker.Base.GetLabels()
	labels["base-app"] = nn.Name

	labeler := maker.MakeLabeler(nn, labels)

	labeler(&dd)

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
			Value: nn.Name + ":32181",
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

	if err = update.Apply(&dd); err != nil {
		return err
	}

	s := core.Service{}
	update, err = maker.Get(nn, &s)
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

	labeler(&s)

	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts

	if err = update.Apply(&s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}
	update, err = maker.Get(nn, &pvc)
	if err != nil {
		return err
	}

	labeler(&pvc)

	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(&pvc); err != nil {
		return err
	}
	return nil
}
