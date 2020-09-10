package makers

import (
	strimzi "cloud.redhat.com/whippoorwill/v2/apis/kafka.strimzi.io/v1beta1"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"

	"k8s.io/apimachinery/pkg/types"
)

//KafkaMaker makes the KafkaConfig object
type KafkaMaker struct {
	*Maker
	config config.KafkaConfig
}

//Make function for the KafkaMaker
func (k *KafkaMaker) Make() error {
	k.config = config.KafkaConfig{}

	var f func() error

	switch k.Base.Spec.Kafka.Provider {
	case "operator":
		f = k.operator
	case "local":
		f = k.local
	}

	if f != nil {
		return f()
	}

	return nil
}

//ApplyConfig for the KafkaMaker
func (k *KafkaMaker) ApplyConfig(c *config.AppConfig) {
	c.Kafka = k.config
}

func (k *KafkaMaker) local() error {
	return nil
}

func (k *KafkaMaker) operator() error {
	if k.App.Spec.KafkaTopics == nil {
		return nil
	}

	k.config.Topics = []config.TopicConfig{}
	k.config.Brokers = []config.BrokerConfig{}

	for _, kafkaTopic := range k.App.Spec.KafkaTopics {
		kRes := strimzi.KafkaTopic{}

		kafkaNamespace := types.NamespacedName{
			Namespace: k.Base.Spec.Kafka.Namespace,
			Name:      kafkaTopic.TopicName,
		}

		err := k.Client.Get(k.Ctx, kafkaNamespace, &kRes)
		update, err := utils.UpdateOrErr(err)

		if err != nil {
			return err
		}

		labels := map[string]string{
			"strimzi.io/cluster": k.Base.Spec.Kafka.ClusterName,
			"iapp":               k.App.GetName(),
			// If we label it with the app name, since app names should be
			// unique? can we use for delete selector?
		}

		kRes.SetName(kafkaTopic.TopicName)
		kRes.SetNamespace(k.Base.Spec.Kafka.Namespace)
		kRes.SetLabels(labels)

		kRes.Spec.Replicas = kafkaTopic.Replicas
		kRes.Spec.Partitions = kafkaTopic.Partitions
		kRes.Spec.Config = kafkaTopic.Config
		err = update.Apply(k.Ctx, k.Client, &kRes)

		if err != nil {
			return err
		}

		k.config.Topics = append(
			k.config.Topics,
			config.TopicConfig{Name: kafkaTopic.TopicName},
		)
	}

	clusterName := types.NamespacedName{
		Namespace: k.Base.Spec.Kafka.Namespace,
		Name:      k.Base.Spec.Kafka.ClusterName,
	}

	kafkaResource := strimzi.Kafka{}
	err := k.Client.Get(k.Ctx, clusterName, &kafkaResource)

	if err != nil {
		return err
	}

	for _, listener := range kafkaResource.Status.Listeners {
		print(listener.Type)
		if listener.Type == "plain" {
			k.config.Brokers = append(
				k.config.Brokers,
				config.BrokerConfig{
					Hostname: listener.Addresses[0].Host,
					Port:     listener.Addresses[0].Port,
				},
			)
		}
	}

	return nil
}
