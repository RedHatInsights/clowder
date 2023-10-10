package kafka

import (
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
)

// KafkaTopic is the resource ident for a KafkaTopic object.
var MSKKafkaTopic = rc.NewSingleResourceIdent(ProvName, "msk_kafka_topic", &strimzi.KafkaTopic{}, rc.ResourceOptions{WriteNow: true})

// MSKKafkaConnect is the resource ident for a KafkaConnect object.
var MSKKafkaConnect = rc.NewSingleResourceIdent(ProvName, "msk_kafka_connect", &strimzi.KafkaConnect{}, rc.ResourceOptions{WriteNow: true})

type mskProvider struct {
	providers.Provider
}

// NewStrimzi returns a new strimzi provider object.
func NewMSK(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CyndiPipeline,
		CyndiAppSecret,
		CyndiHostInventoryAppSecret,
		CyndiConfigMap,
		MSKKafkaTopic,
		MSKKafkaConnect,
	)
	return &mskProvider{Provider: *p}, nil
}

func (s *mskProvider) EnvProvide() error {
	return s.configureBrokers()
}

func (s *mskProvider) Provide(app *crd.ClowdApp) error {
	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	if err := s.processTopics(app, s.Config.Kafka); err != nil {
		return err
	}

	if app.Spec.Cyndi.Enabled {
		err := createCyndiPipeline(s, app, getConnectNamespace(s.Env), getConnectClusterName(s.Env))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *mskProvider) getBootstrapServersString() string {
	strArray := []string{}
	for _, bc := range s.Config.Kafka.Brokers {
		if bc.Port != nil {
			strArray = append(strArray, fmt.Sprintf("%s:%d", bc.Hostname, *bc.Port))
		} else {
			strArray = append(strArray, bc.Hostname)
		}
	}
	return strings.Join(strArray, ",")
}

func (s *mskProvider) getKafkaConfig(broker config.BrokerConfig) *config.KafkaConfig {
	kafkaConfig := &config.KafkaConfig{}
	kafkaConfig.Brokers = []config.BrokerConfig{broker}
	kafkaConfig.Topics = []config.TopicConfig{}

	return kafkaConfig

}

func (s *mskProvider) configureListeners() error {
	var err error
	var secret *core.Secret
	var broker config.BrokerConfig

	secret, err = getSecret(s)
	if err != nil {
		return err
	}

	broker, err = getBrokerConfig(secret)
	if err != nil {
		return err
	}

	s.Config.Kafka = s.getKafkaConfig(broker)

	return nil
}

func (s *mskProvider) configureBrokers() error {
	// Look up Kafka cluster's listeners and configure s.Config.Brokers
	// (we need to know the bootstrap server addresses before provisioning KafkaConnect)
	if err := s.configureListeners(); err != nil {
		clowdErr := errors.Wrap("unable to determine kafka broker addresses", err)
		clowdErr.Requeue = true
		return clowdErr
	}

	if err := configureKafkaConnectCluster(s); err != nil {
		return errors.Wrap("failed to provision kafka connect cluster", err)
	}

	return nil
}

func (s *mskProvider) processTopics(app *crd.ClowdApp, c *config.KafkaConfig) error {
	return processTopics(s, app, c)
}

func (s *mskProvider) getConnectClusterUserName() string {
	return fmt.Sprintf("%s-connect", s.Env.Name)
}

func (s *mskProvider) KafkaTopicName(topic crd.KafkaTopicSpec, namespace string) string {
	if clowderconfig.LoadedConfig.Features.UseComplexStrimziTopicNames {
		return fmt.Sprintf("%s-%s-%s", topic.TopicName, s.Env.Name, namespace)
	}
	return topic.TopicName
}

func (s *mskProvider) KafkaName() string {
	return s.Env.Spec.Providers.Kafka.ClusterAnnotation
}

func (s *mskProvider) KafkaNamespace() string {
	if s.Env.Spec.Providers.Kafka.TopicNamespace == "" {
		return s.Env.Status.TargetNamespace
	}
	return s.Env.Spec.Providers.Kafka.TopicNamespace
}
