package kafka

import (
	"testing"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	"github.com/stretchr/testify/assert"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
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

	assert.NoError(t, err)

	err = ai.EnvProvide()
	assert.NoError(t, err)

	err = ai.Provide(app)
	assert.NoError(t, err)

	assert.Len(t, ai.GetConfig().Kafka.Brokers, 1, "wrong number of brokers")

	broker := ai.GetConfig().Kafka.Brokers[0]

	hostname := "platform-mq-kafka-bootstrap.platform-mq-prod.svc"
	assert.Equal(t, hostname, broker.Hostname, "wrong broker")
	assert.Equal(t, 9092, *broker.Port, "wrong broker port")
	assert.Len(t, ai.GetConfig().Kafka.Topics, 1, "wrong number of topic")

	topic := ai.GetConfig().Kafka.Topics[0]
	assert.Equal(t, topicName, topic.Name, "wrong topic name")
	assert.Equal(t, topicName, topic.RequestedName, "wrong requested topic name")
}
