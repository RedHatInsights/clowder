package kafka

import (
	"fmt"

	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type appInterface struct {
	p.Provider
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
	nn := types.NamespacedName{
		Name:      p.Env.Spec.Kafka.ClusterName,
		Namespace: p.Env.Spec.Kafka.Namespace,
	}

	svc := core.Service{}
	err := p.Client.Get(p.Ctx, nn, &svc)

	if err != nil {
		errors.LogError(p.Ctx, "kafka", errors.New("Cannot find kafka bootstrap service"))
		qualifiedName := fmt.Sprintf("%s:%s", nn.Namespace, nn.Name)
		return nil, &errors.MissingDependencies{
			MissingDeps: map[string][]string{"kafka": {qualifiedName}},
		}
	}

	config := config.KafkaConfig{
		Topics: []config.TopicConfig{},
		Brokers: []config.BrokerConfig{{
			Hostname: fmt.Sprintf("%v-kafka-bootstrap.%v.svc", nn.Name, nn.Namespace),
			Port:     intPtr(9092),
		}},
	}

	kafkaProvider := appInterface{
		Provider: *p,
		Config:   config,
	}

	return &kafkaProvider, nil
}
