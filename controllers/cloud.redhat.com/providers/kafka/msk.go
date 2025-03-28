package kafka

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
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

	dest := crd.NamespacedName{
		Name:      s.Env.Spec.Providers.Kafka.ManagedSecretRef.Name,
		Namespace: s.Env.Spec.TargetNamespace,
	}

	if _, err := s.copyGenericSecret(s.Env, s.Env.Spec.Providers.Kafka.ManagedSecretRef, dest, KafkaManagedSecret); err != nil {
		return err
	}

	if err := s.createConnectSecret(); err != nil {
		return err
	}

	kafkaCASecName := crd.NamespacedName{
		Name:      fmt.Sprintf("%s-cluster-ca-cert", getKafkaName(s.Env)),
		Namespace: getKafkaNamespace(s.Env),
	}

	dest = crd.NamespacedName{
		Name:      fmt.Sprintf("%s-cluster-ca-cert", getKafkaName(s.Env)),
		Namespace: s.Env.Spec.TargetNamespace,
	}

	if _, err := s.copyGenericSecret(s.Env, kafkaCASecName, dest, KafkaManagedSecret); err != nil {
		return err
	}

	return nil
}

func (s *mskProvider) copyGenericSecret(obj object.ClowdObject, source, dest crd.NamespacedName, destIdent rc.ResourceIdent) (bool, error) {
	if source.Namespace == dest.Namespace {
		return true, nil
	}
	sourcePullSecObj := &core.Secret{}
	if err := s.Client.Get(s.Ctx, types.NamespacedName{
		Name:      source.Name,
		Namespace: source.Namespace,
	}, sourcePullSecObj); err != nil {
		return false, err
	}

	newPullSecObj := &core.Secret{}

	newSecNN := types.NamespacedName{
		Name:      dest.Name,
		Namespace: dest.Namespace,
	}

	if err := s.Cache.Create(destIdent, newSecNN, newPullSecObj); err != nil {
		return false, err
	}

	newPullSecObj.Data = sourcePullSecObj.Data
	newPullSecObj.Type = sourcePullSecObj.Type

	labeler := utils.GetCustomLabeler(map[string]string{}, newSecNN, obj)
	labeler(newPullSecObj)

	newPullSecObj.Name = newSecNN.Name
	newPullSecObj.Namespace = newSecNN.Namespace

	if err := s.Cache.Update(destIdent, newPullSecObj); err != nil {
		return false, err
	}
	return false, nil
}

func (s *mskProvider) createConnectSecret() error {
	secName := s.getConnectClusterUserName()

	newPullSecObj := &core.Secret{}

	newSecNN := types.NamespacedName{
		Name:      secName,
		Namespace: s.Env.Status.TargetNamespace,
	}

	if err := s.Cache.Create(KafkaConnectSecret, newSecNN, newPullSecObj); err != nil {
		return err
	}

	newPullSecObj.StringData = map[string]string{
		"password": *s.Config.Kafka.Brokers[0].Sasl.Password,
		"sasl.jaas.config": fmt.Sprintf(
			"org.apache.kafka.common.security.scram.ScramLoginModule required username=\"%s\" password=\"%s\";",
			*s.Config.Kafka.Brokers[0].Sasl.Username,
			*s.Config.Kafka.Brokers[0].Sasl.Password,
		),
	}
	newPullSecObj.Type = core.SecretTypeOpaque

	labeler := utils.GetCustomLabeler(map[string]string{}, newSecNN, s.Env)
	labeler(newPullSecObj)

	newPullSecObj.Name = newSecNN.Name
	newPullSecObj.Namespace = newSecNN.Namespace

	if err := s.Cache.Update(KafkaConnectSecret, newPullSecObj); err != nil {
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

	replicas := 3
	if s.Env.Spec.Providers.Kafka.KafkaConnectReplicaCount != 0 {
		replicas = s.Env.Spec.Providers.Kafka.KafkaConnectReplicaCount
	}

	connectConfig := genericConfig{
		"config.storage.replication.factor":       strconv.Itoa(replicas),
		"config.storage.topic":                    fmt.Sprintf("%v-connect-cluster-configs", s.Env.Name),
		"connector.client.config.override.policy": "All",
		"group.id":                          "connect-cluster",
		"offset.storage.replication.factor": strconv.Itoa(replicas),
		"offset.storage.topic":              fmt.Sprintf("%v-connect-cluster-offsets", s.Env.Name),
		"status.storage.replication.factor": strconv.Itoa(replicas),
		"status.storage.topic":              fmt.Sprintf("%v-connect-cluster-status", s.Env.Name),
	}

	byteData, err := json.Marshal(connectConfig)
	if err != nil {
		return err
	}
	return config.UnmarshalJSON(byteData)
}

func (s *mskProvider) getKafkaConfig(brokers []config.BrokerConfig) *config.KafkaConfig {
	kafkaConfig := &config.KafkaConfig{}
	kafkaConfig.Brokers = brokers
	kafkaConfig.Topics = []config.TopicConfig{}

	return kafkaConfig

}

func (s *mskProvider) configureListeners() error {
	var err error
	var secret *core.Secret
	var brokers []config.BrokerConfig

	secret, err = getSecret(s)
	if err != nil {
		return err
	}

	if _, err := s.HashCache.CreateOrUpdateObject(secret, true); err != nil {
		return err
	}

	if err := s.HashCache.AddClowdObjectToObject(s.Env, secret); err != nil {
		return err
	}

	brokers, err = getBrokerConfig(secret)
	if err != nil {
		return err
	}

	s.Config.Kafka = s.getKafkaConfig(brokers)

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

func (s *mskProvider) KafkaTopicName(topic crd.KafkaTopicSpec, _ ...string) (string, error) {
	return fmt.Sprintf("%s-%s", s.Env.Name, topic.TopicName), nil
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
