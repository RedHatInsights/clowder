package kafka

import (
	"encoding/json"
	"fmt"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	core "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
)

// KafkaManagedSecret is the resource ident for the MSK user secret object.
var KafkaManagedSecret = rc.NewMultiResourceIdent(ProvName, "kafka_managed_secret", &core.Secret{})

// KafkaConnectSecret is the resource ident for a MSK Connect secret object.
var KafkaConnectSecret = rc.NewMultiResourceIdent(ProvName, "kafka_connect_secret", &core.Secret{})

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
		KafkaTopic,
		KafkaConnect,
		KafkaManagedSecret,
		KafkaConnectSecret,
	)
	return &mskProvider{Provider: *p}, nil
}

func (s *mskProvider) EnvProvide() error {
	s.Config = &config.AppConfig{
		Kafka: &config.KafkaConfig{},
	}

	if err := s.configureBrokers(); err != nil {
		return err
	}

	namespaceList, err := s.Env.GetNamespacesInEnv(s.Ctx, s.Client)
	if err != nil {
		return err
	}

	for _, namespace := range namespaceList {
		if err := s.copyManagedSecret(namespace); err != nil {
			return err
		}
		if err := s.copyConnectSecret(namespace); err != nil {
			return err
		}
	}

	return nil
}

func (s *mskProvider) copyManagedSecret(namespace string) error {
	srcSecretRef := types.NamespacedName{
		Name:      s.Env.Spec.Providers.Kafka.ManagedSecretRef.Name,
		Namespace: s.Env.Spec.Providers.Kafka.ManagedSecretRef.Namespace,
	}
	dstSecretRef := types.NamespacedName{
		Name:      srcSecretRef.Name,
		Namespace: namespace,
	}
	sec, err := utils.CopySecret(s.Ctx, s.Client, srcSecretRef, dstSecretRef)

	if err != nil {
		return err
	}

	if err = s.Cache.Create(KafkaManagedSecret, dstSecretRef, sec); err != nil {
		s.Log.Error(err, "Failed to add managed secret to cache")
		return err
	}
	return nil
}

func (s *mskProvider) copyConnectSecret(namespace string) error {
	secName := s.getConnectClusterUserName()

	srcSecretRef := types.NamespacedName{
		Name:      secName,
		Namespace: s.Env.Spec.Providers.Kafka.ManagedSecretRef.Namespace,
	}
	dstSecretRef := types.NamespacedName{
		Name:      secName,
		Namespace: namespace,
	}

	sec, err := utils.CopySecret(s.Ctx, s.Client, srcSecretRef, dstSecretRef)
	if err != nil {
		return err
	}

	if err = s.Cache.Create(KafkaConnectSecret, dstSecretRef, sec); err != nil {
		s.Log.Error(err, "Failed to add managed secret to cache")
		return err
	}
	return nil
}

func (s *mskProvider) Provide(app *crd.ClowdApp) error {
	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	s.Config.Kafka = &config.KafkaConfig{}
	s.Config.Kafka.Brokers = []config.BrokerConfig{}
	s.Config.Kafka.Topics = []config.TopicConfig{}

	if err := s.configureListeners(); err != nil {
		return err
	}

	if err := processTopics(s, app); err != nil {
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

type genericConfig map[string]string

func (s mskProvider) connectConfig(config *apiextensions.JSON) error {

	connectConfig := genericConfig{
		"config.storage.replication.factor":       "1",
		"config.storage.topic":                    fmt.Sprintf("%v-connect-cluster-configs", s.Env.Name),
		"connector.client.config.override.policy": "All",
		"group.id":                          "connect-cluster",
		"offset.storage.replication.factor": "1",
		"offset.storage.topic":              fmt.Sprintf("%v-connect-cluster-offsets", s.Env.Name),
		"status.storage.replication.factor": "1",
		"status.storage.topic":              fmt.Sprintf("%v-connect-cluster-status", s.Env.Name),
	}

	byteData, err := json.Marshal(connectConfig)
	if err != nil {
		return err
	}
	return config.UnmarshalJSON(byteData)
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

func (s *mskProvider) getKafkaConnectTrustedCertSecretName() (string, error) {
	secRef, err := getSecretRef(s)
	if err != nil {
		return "", err
	}
	return secRef.Name, nil
}

func (s *mskProvider) getConnectClusterUserName() string {
	return *s.Config.Kafka.Brokers[0].Sasl.Username
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
