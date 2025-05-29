package kafka

import (
	"fmt"
	"strconv"
	"strings"

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
	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	var err error
	var secret *core.Secret
	var brokers []config.BrokerConfig

	secret, err = k.getSecret()
	if err != nil {
		return err
	}

	brokers, err = k.getBrokerConfig(secret)
	if err != nil {
		return err
	}

	k.Config.Kafka = k.getKafkaConfig(brokers, app)

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

func (k *managedKafkaProvider) destructureSecret(secret *core.Secret) (int, string, string, string, []string, string, string, error) {
	port, err := strconv.Atoi(string(secret.Data["port"]))
	if err != nil {
		return 0, "", "", "", []string{}, "", "", err
	}
	password := string(secret.Data["password"])
	username := string(secret.Data["username"])
	hostname := string(secret.Data["hostname"])
	cacert := ""
	if val, ok := secret.Data["cacert"]; ok {
		cacert = string(val)
	}
	saslMechanism := "PLAIN"
	if val, ok := secret.Data["saslMechanism"]; ok {
		saslMechanism = string(val)
	}
	hostnames := []string{}
	if val, ok := secret.Data["hostnames"]; ok {
		// 'hostnames' key is expected to be a comma,separated,list of broker hostnames
		hostnames = strings.Split(string(val), ",")
	}
	return int(port), password, username, hostname, hostnames, cacert, saslMechanism, nil
}

func (k *managedKafkaProvider) getBrokerConfig(secret *core.Secret) ([]config.BrokerConfig, error) {
	brokers := []config.BrokerConfig{}

	port, password, username, hostname, hostnames, cacert, saslMechanism, err := k.destructureSecret(secret)
	if err != nil {
		return brokers, err
	}

	if len(hostnames) == 0 {
		// if there is no 'hostnames' key found, fall back to using 'hostname' key
		hostnames = append(hostnames, hostname)
	}

	saslType := config.BrokerConfigAuthtypeSasl

	broker := config.BrokerConfig{}
	for _, hostname := range hostnames {
		broker.Hostname = string(hostname)
		broker.Port = &port
		broker.Authtype = &saslType
		if cacert != "" {
			broker.Cacert = &cacert
		}
		broker.Sasl = &config.KafkaSASLConfig{
			Password:         &password,
			Username:         &username,
			SecurityProtocol: utils.StringPtr("SASL_SSL"),
			SaslMechanism:    utils.StringPtr(saslMechanism),
		}
		broker.SecurityProtocol = utils.StringPtr("SASL_SSL")
		brokers = append(brokers, broker)
	}

	return brokers, nil
}

func (k *managedKafkaProvider) getKafkaConfig(brokers []config.BrokerConfig, app *crd.ClowdApp) *config.KafkaConfig {
	kafkaConfig := &config.KafkaConfig{}
	kafkaConfig.Brokers = brokers
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

	_, err = k.HashCache.CreateOrUpdateObject(secret, true)
	if err != nil {
		return nil, err
	}

	if err = k.HashCache.AddClowdObjectToObject(k.Env, secret); err != nil {
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
		return nullName, errors.NewClowderError("no secret ref defined for managed Kafka")
	}
	return secretRef, nil
}
