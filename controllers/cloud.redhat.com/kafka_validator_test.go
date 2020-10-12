package controllers

import (
	"fmt"

	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"
)

func LocalKafkaValidator(input groupOfCRs) error {
	d := apps.Deployment{}
	s := core.Service{}
	p := core.PersistentVolumeClaim{}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-%v", input.Env.Name, "kafka"),
		Namespace: input.Env.Spec.Namespace,
	}

	err := fetchWithDefaults(nn, &d)
	if err != nil {
		return err
	}
	err = fetchWithDefaults(nn, &s)
	if err != nil {
		return err
	}
	err = fetchWithDefaults(nn, &p)
	if err != nil {
		return err
	}

	return nil
}

func OperatorKafkaValidator(input groupOfCRs) error {
	for _, app := range input.Apps {
		if app.Spec.KafkaTopics != nil {
			for _, topic := range app.Spec.KafkaTopics {
				nn := types.NamespacedName{
					Namespace: input.Env.Spec.Kafka.Namespace,
					Name:      topic.TopicName,
				}
				ktopic := &strimzi.KafkaTopic{}
				err := fetchWithDefaults(nn, ktopic)
				if err != nil {
					return err
				}
				if topic.Replicas != ktopic.Spec.Replicas {
					return fmt.Errorf("Topic %v didn't not have correct replicas, should be %v, was %v", topic.TopicName, topic.Replicas, ktopic.Spec.Replicas)
				}
				if topic.Partitions != ktopic.Spec.Partitions {
					return fmt.Errorf("Topic %v didn't not have correct partitions, should be %v, was %v", topic.TopicName, topic.Partitions, ktopic.Spec.Partitions)
				}
				if topic.Config != nil {
					retentionMS, ok := topic.Config["retention.ms"]
					if ok {
						if retentionMS != ktopic.Spec.Config["retention.ms"] {
							return fmt.Errorf("Topic %v config for retention.ms didn't not have correct value, should be %v, was %v", topic.TopicName, retentionMS, ktopic.Spec.Config["retention.ms"])
						}
					}
				}
			}
		}
	}

	return nil
}

func KafkaValidator(input groupOfCRs) error {
	var fn func(input groupOfCRs) error
	switch input.Env.Spec.Kafka.Provider {
	case "local":
		fn = LocalKafkaValidator
	case "operator":
		fn = OperatorKafkaValidator
	}
	return fn(input)
}
