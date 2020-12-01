package kafka

import (
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

func TestAppInterface(t *testing.T) {
	clusterName, ns, topicName := "platform-mq", "platform-mq-prod", "ingress"
	pr := p.Provider{
		Env: &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Kafka: crd.KafkaConfig{
						ClusterName: clusterName,
						Namespace:   ns,
					},
				},
			},
		},
	}

	app := &crd.ClowdApp{
		Spec: crd.ClowdAppSpec{
			KafkaTopics: []strimzi.KafkaTopicSpec{{
				TopicName: topicName,
			}},
		},
	}

	ai, err := NewAppInterface(&pr)

	if err != nil {
		t.Error(err)
	}

	ai.CreateTopics(app)

	c := config.AppConfig{}
	ai.Configure(&c)

	if len(c.Kafka.Brokers) != 1 {
		t.Errorf("Wrong number of brokers %v; expected 1", len(c.Kafka.Brokers))
	}

	broker := c.Kafka.Brokers[0]

	hostname := "platform-mq-kafka-bootstrap.platform-mq-prod.svc"
	if broker.Hostname != hostname {
		t.Errorf("Wrong broker %v; expected %v", broker.Hostname, hostname)
	}

	if *broker.Port != 9092 {
		t.Errorf("Wrong broker port %v; expected %v", broker.Port, 9092)
	}

	if len(c.Kafka.Topics) != 1 {
		t.Errorf("Wrong number of topic %v; expected 1", len(c.Kafka.Topics))
	}

	topic := c.Kafka.Topics[0]

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
