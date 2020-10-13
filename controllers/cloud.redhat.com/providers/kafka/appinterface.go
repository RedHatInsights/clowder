package kafka

import (
	"context"
	"fmt"

	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type appInterface struct {
	p.Provider
	Config config.KafkaConfig
}

func (a *appInterface) Configure(config *config.AppConfig) {
	config.Kafka = &a.Config
}

func (a *appInterface) CreateTopic(nn types.NamespacedName, topic *strimzi.KafkaTopicSpec) error {
	topicName := types.NamespacedName{
		Namespace: a.Env.Spec.Kafka.Namespace,
		Name:      topic.TopicName,
	}

	err := validateKafkaTopic(a.Ctx, a.Client, topicName)

	if err != nil {
		return err
	}

	a.Config.Topics = append(
		a.Config.Topics,
		config.TopicConfig{
			Name:          topic.TopicName,
			RequestedName: topic.TopicName,
		},
	)

	return nil
}

func validateKafkaTopic(ctx context.Context, cl client.Client, nn types.NamespacedName) error {
	if cl == nil {
        // Don't validate topics from within test suite
		return nil
	}

	t := strimzi.KafkaTopic{}
	err := cl.Get(ctx, nn, &t)

	if err != nil {
		return &errors.MissingDependencies{
			MissingDeps: map[string][]string{"topics": {nn.Name}},
		}
	}

	return nil
}

func validateBrokerService(ctx context.Context, cl client.Client, nn types.NamespacedName) error {
	if cl == nil {
        // Don't validate brokers from within test suite
		return nil
	}

	svc := core.Service{}
	err := cl.Get(ctx, nn, &svc)

	if err != nil {
		errors.LogError(ctx, "kafka", errors.New("Cannot find kafka bootstrap service"))
		qualifiedName := fmt.Sprintf("%s:%s", nn.Namespace, nn.Name)
		return &errors.MissingDependencies{
			MissingDeps: map[string][]string{"kafka": {qualifiedName}},
		}
	}

	return nil
}

func NewAppInterface(p *p.Provider) (KafkaProvider, error) {
	nn := types.NamespacedName{
		Name:      p.Env.Spec.Kafka.ClusterName,
		Namespace: p.Env.Spec.Kafka.Namespace,
	}

	err := validateBrokerService(p.Ctx, p.Client, nn)

	if err != nil {
		return nil, err
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
