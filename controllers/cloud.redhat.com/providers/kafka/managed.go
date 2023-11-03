package kafka

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	core "k8s.io/api/core/v1"
)

type managedKafkaProvider struct {
	providers.Provider
}

// NewNoneKafka returns a new non kafka provider object.
func NewManagedKafka(p *providers.Provider) (providers.ClowderProvider, error) {
	return &managedKafkaProvider{Provider: *p}, nil
}

func (k *managedKafkaProvider) EnvProvide() error {
	return nil
}

func (k *managedKafkaProvider) Provide(app *crd.ClowdApp) error {
	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	var err error
	var secret *core.Secret
	var broker config.BrokerConfig

	secret, err = getSecret(k)
	if err != nil {
		return err
	}

	broker, err = getBrokerConfig(secret)
	if err != nil {
		return err
	}

	k.Config.Kafka = k.getKafkaConfig(broker, app)

	return nil
}

func (k *managedKafkaProvider) appendTopic(topic crd.KafkaTopicSpec, kafkaConfig *config.KafkaConfig) {

	topicName := topic.TopicName

	if k.Env.Spec.Providers.Kafka.ManagedPrefix != "" {
		topicName = fmt.Sprintf("%s%s", k.Env.Spec.Providers.Kafka.ManagedPrefix, topicName)
	}

	kafkaConfig.Topics = append(
		kafkaConfig.Topics,
		config.TopicConfig{
			Name:          topicName,
			RequestedName: topic.TopicName,
		},
	)
}

func (k *managedKafkaProvider) getKafkaConfig(broker config.BrokerConfig, app *crd.ClowdApp) *config.KafkaConfig {
	kafkaConfig := &config.KafkaConfig{}
	kafkaConfig.Brokers = []config.BrokerConfig{broker}
	kafkaConfig.Topics = []config.TopicConfig{}

	for _, topic := range app.Spec.KafkaTopics {
		k.appendTopic(topic, kafkaConfig)
	}

	return kafkaConfig

}
