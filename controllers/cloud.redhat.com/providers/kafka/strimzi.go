package kafka

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"k8s.io/apimachinery/pkg/types"
)

type strimziProvider struct {
	p.Provider
	Config config.KafkaConfig
}

func (s *strimziProvider) Configure(config *config.AppConfig) {
	config.Kafka = &s.Config
}

func (s *strimziProvider) configureBrokers() error {
	clusterName := types.NamespacedName{
		Namespace: s.Env.Spec.Kafka.Namespace,
		Name:      s.Env.Spec.Kafka.ClusterName,
	}

	kafkaResource := strimzi.Kafka{}

	if _, err := utils.UpdateOrErr(s.Client.Get(s.Ctx, clusterName, &kafkaResource)); err != nil {
		return err
	}

	for _, listener := range kafkaResource.Status.Listeners {
		if listener.Type == "plain" {
			bc := config.BrokerConfig{
				Hostname: listener.Addresses[0].Host,
			}
			port := listener.Addresses[0].Port
			if port != nil {
				p := int(*port)
				bc.Port = &p
			}
			s.Config.Brokers = append(s.Config.Brokers, bc)
		}
	}

	return nil
}

func NewStrimzi(p *p.Provider) (KafkaProvider, error) {
	kafkaProvider := &strimziProvider{
		Provider: *p,
		Config: config.KafkaConfig{
			Brokers: []config.BrokerConfig{},
		},
	}

	return kafkaProvider, kafkaProvider.configureBrokers()
}

func (s *strimziProvider) CreateTopic(nn types.NamespacedName, topic *strimzi.KafkaTopicSpec) error {
	s.Config.Topics = []config.TopicConfig{}

	appList := crd.ClowdAppList{}
	err := s.Client.List(s.Ctx, &appList)

	if err != nil {
		return errors.Wrap("Topic creation failed: Error listing apps", err)
	}

	kRes := strimzi.KafkaTopic{}

	topicName := fmt.Sprintf("%s-%s-%s", topic.TopicName, s.Env.Name, nn.Namespace)

	update, err := utils.UpdateOrErr(s.Client.Get(s.Ctx, types.NamespacedName{
		Namespace: s.Env.Spec.Kafka.Namespace,
		Name:      topicName,
	}, &kRes))

	if err != nil {
		return err
	}

	labels := p.Labels{
		"strimzi.io/cluster": s.Env.Spec.Kafka.ClusterName,
		"app":                nn.Name,
		// If we label it with the app name, since app names should be
		// unique? can we use for delete selector?
	}

	kRes.SetName(topicName)
	kRes.SetNamespace(s.Env.Spec.Kafka.Namespace)
	kRes.SetLabels(labels)

	kRes.Spec.Replicas = topic.Replicas
	kRes.Spec.Partitions = topic.Partitions
	kRes.Spec.Config = topic.Config

	// This can be improved from an efficiency PoV
	// Loop through all key/value pairs in the config

	retentionMsValList := []int{}
	retentionBytesValList := []int{}
	minCompactionLagMsValList := []int{}
	cleanupPolicyValList := []string{}

	if kRes.Spec.Config != nil {
		if kRes.Spec.Config.RetentionMs != 0 {
			retentionMsValList = append(retentionMsValList, kRes.Spec.Config.RetentionMs)
		}
		if kRes.Spec.Config.RetentionBytes != 0 {
			retentionBytesValList = append(retentionBytesValList, kRes.Spec.Config.RetentionBytes)
		}
		if kRes.Spec.Config.MinCompactionLagMs != 0 {
			minCompactionLagMsValList = append(minCompactionLagMsValList, kRes.Spec.Config.MinCompactionLagMs)
		}
		if kRes.Spec.Config.CleanupPolicy != "" {
			cleanupPolicyValList = append(cleanupPolicyValList, kRes.Spec.Config.CleanupPolicy)
		}
	}
	for _, res := range appList.Items {
		if res.ObjectMeta.Name == nn.Name {
			continue
		}
		if res.ObjectMeta.Namespace != nn.Namespace {
			continue
		}
		if res.Spec.KafkaTopics != nil {
			for _, appTopic := range res.Spec.KafkaTopics {
				if appTopic.TopicName != topic.TopicName {
					continue
				}
				if appTopic.Config != nil {
					if appTopic.Config.RetentionMs != 0 {
						retentionMsValList = append(retentionMsValList, appTopic.Config.RetentionMs)
					}
					if appTopic.Config.RetentionBytes != 0 {
						retentionBytesValList = append(retentionBytesValList, appTopic.Config.RetentionBytes)
					}
					if appTopic.Config.MinCompactionLagMs != 0 {
						minCompactionLagMsValList = append(minCompactionLagMsValList, appTopic.Config.MinCompactionLagMs)
					}
					if appTopic.Config.CleanupPolicy != "" {
						cleanupPolicyValList = append(cleanupPolicyValList, appTopic.Config.CleanupPolicy)
					}
				}
			}
		}
	}

	newConfig := &strimzi.KafkaConfig{}

	if len(retentionMsValList) > 0 {
		retentionMsFinalValue, err := utils.IntMax(retentionMsValList)
		if err != nil {
			return err
		}
		newConfig.RetentionMs = retentionMsFinalValue
	}
	if len(retentionBytesValList) > 0 {
		retentionBytesFinalValue, err := utils.IntMax(retentionBytesValList)
		if err != nil {
			return err
		}
		newConfig.RetentionBytes = retentionBytesFinalValue
	}
	if len(minCompactionLagMsValList) > 0 {
		minCompactionLagMsFinalValue, err := utils.IntMax(minCompactionLagMsValList)
		if err != nil {
			return err
		}
		newConfig.MinCompactionLagMs = minCompactionLagMsFinalValue
	}
	if len(cleanupPolicyValList) > 0 {
		cleanupPolicyFinalValue, err := utils.ListMerge(cleanupPolicyValList)
		if err != nil {
			return err
		}
		newConfig.CleanupPolicy = cleanupPolicyFinalValue
	}

	kRes.Spec.Config = newConfig

	if err = update.Apply(s.Ctx, s.Client, &kRes); err != nil {
		return err
	}

	s.Config.Topics = append(
		s.Config.Topics,
		config.TopicConfig{Name: topicName, RequestedName: topic.TopicName},
	)

	return nil
}
