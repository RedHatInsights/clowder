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

func (k *managedKafkaProvider) EnvProvide() error {
	return nil
}

func (k *managedKafkaProvider) Provide(app *crd.ClowdApp) error {
	var err error
	var secret *core.Secret
	var broker config.BrokerConfig

	secret, err = k.getSecret()
	if err != nil {
		return err
	}

	broker, err = k.getBrokerConfig(secret)
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

func (k *managedKafkaProvider) destructureSecret(secret *core.Secret) (int, string, string, string, string, error) {
	port, err := strconv.Atoi(string(secret.Data["port"]))
	if err != nil {
		return 0, "", "", "", "", err
	}
	password := string(secret.Data["password"])
	username := string(secret.Data["username"])
	hostname := string(secret.Data["hostname"])
	cacert := ""
	if val, ok := secret.Data["cacert"]; ok {
		cacert = string(val)
	}
	return port, password, username, hostname, cacert, nil
}

func (k *managedKafkaProvider) getBrokerConfig(secret *core.Secret) (config.BrokerConfig, error) {
	broker := config.BrokerConfig{}

	port, password, username, hostname, cacert, err := k.destructureSecret(secret)
	if err != nil {
		return broker, err
	}

	saslType := config.BrokerConfigAuthtypeSasl

	broker.Hostname = hostname
	broker.Port = &port
	broker.Authtype = &saslType
	if cacert != "" {
		broker.Cacert = &cacert
	}
	broker.Sasl = &config.KafkaSASLConfig{
		Password:         &password,
		Username:         &username,
		SecurityProtocol: utils.StringPtr("SASL_SSL"),
		SaslMechanism:    utils.StringPtr("PLAIN"),
	}

	return broker, nil
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

func (k *managedKafkaProvider) getSecret() (*core.Secret, error) {
	secretRef, err := k.getSecretRef()
	if err != nil {
		return nil, err
	}

	secret := &core.Secret{}

	if err = k.Client.Get(k.Ctx, secretRef, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func (k *managedKafkaProvider) getSecretRef() (types.NamespacedName, error) {
	secretRef := types.NamespacedName{
		Name:      k.Env.Spec.Providers.Kafka.ManagedSecretRef.Name,
		Namespace: k.Env.Spec.Providers.Kafka.ManagedSecretRef.Namespace,
	}
	nullName := types.NamespacedName{}
	if secretRef == nullName {
		return nullName, errors.New("no secret ref defined for managed Kafka")
	}
	return secretRef, nil
}
