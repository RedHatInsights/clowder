package kafka

import (
	"fmt"

	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"k8s.io/apimachinery/pkg/types"
)

type appInterface struct {
	providers.Provider
	Config config.KafkaConfig
}

func (a *appInterface) Configure(config *config.AppConfig) {
	config.Kafka = &a.Config
}

func (a *appInterface) CreateTopic(nn types.NamespacedName, topic *strimzi.KafkaTopicSpec) error {
	a.Config.Topics = append(
		a.Config.Topics,
		config.TopicConfig{
			Name:          topic.TopicName,
			RequestedName: topic.TopicName,
		},
	)

	return nil
}

func NewAppInterface(p *p.Provider) (KafkaProvider, error) {
	config := config.KafkaConfig{
		Topics: []config.TopicConfig{},
		Brokers: []config.BrokerConfig{{
			Hostname: fmt.Sprintf(
				"%v-kafka-bootstrap.%v.svc",
				p.Env.Spec.Kafka.ClusterName,
				p.Env.Spec.Kafka.Namespace,
			),
			Port: intPtr(9092),
		}},
	}

	kafkaProvider := appInterface{
		Provider: *p,
		Config:   config,
	}

	return &kafkaProvider, nil
}
