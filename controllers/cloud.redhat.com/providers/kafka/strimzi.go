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

var conversionMap = map[string]func([]string) (string, error){
	"retention.ms":          utils.IntMax,
	"retention.bytes":       utils.IntMax,
	"min.compaction.lag.ms": utils.IntMax,
	"cleanup.policy":        utils.ListMerge,
}

type strimziProvider struct {
	p.Provider
	Config config.KafkaConfig
}

func (s *strimziProvider) Configure(config *config.AppConfig) {
	config.Kafka = &s.Config
}

func (s *strimziProvider) configureBrokers() error {
	clusterName := types.NamespacedName{
		Namespace: s.Env.Spec.Providers.Kafka.Namespace,
		Name:      s.Env.Spec.Providers.Kafka.ClusterName,
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
		Namespace: s.Env.Spec.Providers.Kafka.Namespace,
		Name:      topicName,
	}, &kRes))

	if err != nil {
		return err
	}

	labels := p.Labels{
		"strimzi.io/cluster": s.Env.Spec.Providers.Kafka.ClusterName,
		"app":                nn.Name,
		// If we label it with the app name, since app names should be
		// unique? can we use for delete selector?
	}

	kRes.SetName(topicName)
	kRes.SetNamespace(s.Env.Spec.Providers.Kafka.Namespace)
	kRes.SetLabels(labels)

	kRes.Spec.Replicas = topic.Replicas
	kRes.Spec.Partitions = topic.Partitions
	kRes.Spec.Config = topic.Config

	newConfig := make(map[string]string)

	// This can be improved from an efficiency PoV
	// Loop through all key/value pairs in the config
	for key, value := range kRes.Spec.Config {
		valList := []string{value}
		for _, res := range appList.Items {
			if res.ObjectMeta.Name == nn.Name {
				continue
			}
			if res.ObjectMeta.Namespace != nn.Namespace {
				continue
			}
			if res.Spec.KafkaTopics != nil {
				for _, topic := range res.Spec.KafkaTopics {
					if topic.Config != nil {
						if val, ok := topic.Config[key]; ok {
							valList = append(valList, val)
						}
					}
				}
			}
		}
		f, ok := conversionMap[key]
		if ok {
			out, _ := f(valList)
			newConfig[key] = out
		} else {
			return errors.New(fmt.Sprintf("no conversion type for %s", key))
		}
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
