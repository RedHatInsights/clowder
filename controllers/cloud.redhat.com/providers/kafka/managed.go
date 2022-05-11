package kafka

import (
	"fmt"
	"strconv"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type managedKafkaProvider struct {
	providers.Provider
}

// NewNoneKafka returns a new non kafka provider object.
func NewManagedKafka(p *providers.Provider) (providers.ClowderProvider, error) {
	return &managedKafkaProvider{Provider: *p}, nil
}

func (k *managedKafkaProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	secretRef := types.NamespacedName{
		Name:      k.Env.Spec.Providers.Kafka.ManagedSecretRef.Name,
		Namespace: k.Env.Spec.Providers.Kafka.ManagedSecretRef.Namespace,
	}

	nullName := types.NamespacedName{}

	if secretRef == nullName {
		return errors.New("no secret ref defined for managed Kafka")
	}

	s := &core.Secret{}

	if err := k.Client.Get(k.Ctx, secretRef, s); err != nil {
		return err
	}

	var port int
	var err error

	if port, err = strconv.Atoi(string(s.Data["port"])); err != nil {
		return err
	}

	kafkaConfig := config.KafkaConfig{}

	password := string(s.Data["password"])
	username := string(s.Data["username"])

	broker := config.BrokerConfig{
		Hostname: string(s.Data["hostname"]),
		Port:     &port,
		Sasl: &config.KafkaSASLConfig{
			Password:         &password,
			Username:         &username,
			SecurityProtocol: utils.StringPtr("SASL_SSL"),
			SaslMechanism:    utils.StringPtr("PLAIN"),
		},
	}

	kafkaConfig.Brokers = []config.BrokerConfig{broker}

	for _, topic := range app.Spec.KafkaTopics {

		topicName := topic.TopicName

		if k.Env.Spec.Providers.Kafka.ManagedPrefix != "" {
			topicName = fmt.Sprintf("%s-%s", k.Env.Spec.Providers.Kafka.ManagedPrefix, topicName)
		}

		kafkaConfig.Topics = append(
			kafkaConfig.Topics,
			config.TopicConfig{
				Name:          topicName,
				RequestedName: topic.TopicName,
			},
		)
	}

	c.Kafka = &kafkaConfig

	return nil
}
