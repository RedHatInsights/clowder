package kafka

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

func TestAppInterface(t *testing.T) {
	clusterName, ns, topicName := "platform-mq", "platform-mq-prod", "ingress"
	pr := providers.Provider{
		Env: &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Kafka: crd.KafkaConfig{
						Mode: "app-interface",
						Cluster: crd.KafkaClusterConfig{
							Name:      clusterName,
							Namespace: ns,
						},
					},
				},
			},
		},
		Config: &config.AppConfig{
			Kafka: &config.KafkaConfig{
				Brokers: []config.BrokerConfig{{
					Hostname: "platform-mq-kafka-bootstrap.platform-mq-prod.svc",
					Port:     utils.IntPtr(9092),
				}},
				Topics: []config.TopicConfig{},
			},
		},
	}

	app := &crd.ClowdApp{
		Spec: crd.ClowdAppSpec{
			KafkaTopics: []crd.KafkaTopicSpec{{
				TopicName: topicName,
			}},
		},
	}

	ai, err := NewAppInterface(&pr)

	if err != nil {
		t.Error(err)
	}

	err = ai.EnvProvide()
	if err != nil {
		t.Error(err)
	}

	err = ai.Provide(app)
	if err != nil {
		t.Error(err)
	}

	if len(ai.GetConfig().Kafka.Brokers) != 1 {
		t.Errorf("Wrong number of brokers %v; expected 1", len(ai.GetConfig().Kafka.Brokers))
	}

	broker := ai.GetConfig().Kafka.Brokers[0]

	hostname := "platform-mq-kafka-bootstrap.platform-mq-prod.svc"
	if broker.Hostname != hostname {
		t.Errorf("Wrong broker %v; expected %v", broker.Hostname, hostname)
	}

	if *broker.Port != 9092 {
		t.Errorf("Wrong broker port %v; expected %v", broker.Port, 9092)
	}

	if len(ai.GetConfig().Kafka.Topics) != 1 {
		t.Errorf("Wrong number of topic %v; expected 1", len(ai.GetConfig().Kafka.Topics))
	}

	topic := ai.GetConfig().Kafka.Topics[0]

	if topic.Name != topicName {
		t.Errorf("Wrong topic name %v; expected %v", topic.Name, topicName)
	}

	if topic.RequestedName != topicName {
		t.Errorf(
			"Wrong requested topic name %v; expected %v",
			topic.RequestedName,
			topicName,
		)
	}
}
