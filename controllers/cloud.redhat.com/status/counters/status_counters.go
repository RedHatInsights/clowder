package status

import (
	"context"
	"fmt"
	"sort"
	"strings"

	statusTypes "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/status/types"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func kafkaStatusChecker(kafka strimzi.Kafka) bool {
	// nil checks needed since these are all pointers in strimzi-client-go
	if kafka.Status == nil {
		return false
	}

	if kafka.Status.ObservedGeneration != nil && kafka.Generation > int64(*kafka.Status.ObservedGeneration) {
		// The status on this resource needs to update
		return false
	}

	for _, condition := range kafka.Status.Conditions {
		if condition.Type != nil && *condition.Type == "Ready" && condition.Status != nil && *condition.Status == "True" {
			return true
		}
	}

	return false
}

func kafkaTopicStatusChecker(kafka strimzi.KafkaTopic) bool {
	// nil checks needed since these are all pointers in strimzi-client-go
	if kafka.Status == nil {
		return false
	}

	if kafka.Status.ObservedGeneration != nil && kafka.Generation > int64(*kafka.Status.ObservedGeneration) {
		// The status on this resource needs to update
		return false
	}

	for _, condition := range kafka.Status.Conditions {
		if condition.Type != nil && *condition.Type == "Ready" && condition.Status != nil && *condition.Status == "True" {
			return true
		}
	}

	return false
}

func kafkaConnectStatusChecker(kafkaConnect strimzi.KafkaConnect) bool {
	// nil checks needed since these are all pointers in strimzi-client-go
	if kafkaConnect.Status == nil {
		return false
	}

	if kafkaConnect.Status.ObservedGeneration != nil && kafkaConnect.Generation > int64(*kafkaConnect.Status.ObservedGeneration) {
		// The status on this resource needs to update
		return false
	}

	for _, condition := range kafkaConnect.Status.Conditions {
		if condition.Type != nil && *condition.Type == "Ready" && condition.Status != nil && *condition.Status == "True" {
			return true
		}
	}

	return false
}

func CountKafkas(ctx context.Context, pClient client.Client, statusSource statusTypes.StatusSource, namespaces []string) (int32, int32, string, error) {
	var managedKafkas int32
	var readyKafka int32
	var brokenKafkas []string
	var msg = ""

	kafkas := []strimzi.Kafka{}
	for _, namespace := range namespaces {
		opts := []client.ListOption{
			client.InNamespace(namespace),
		}
		tmpKafkas := strimzi.KafkaList{}
		err := pClient.List(ctx, &tmpKafkas, opts...)
		if err != nil {
			return 0, 0, "", err
		}
		kafkas = append(kafkas, tmpKafkas.Items...)
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, kafka := range kafkas {
		for _, owner := range kafka.GetOwnerReferences() {
			if owner.UID == statusSource.GetUID() {
				managedKafkas++
				if ok := kafkaStatusChecker(kafka); ok {
					readyKafka++
				} else {
					brokenKafkas = append(brokenKafkas, fmt.Sprintf("%s/%s", kafka.Name, kafka.Namespace))
				}
				break
			}
		}
	}

	if len(brokenKafkas) > 0 {
		msg = fmt.Sprintf("broken kafkas: [%s]", strings.Join(brokenKafkas, ", "))
	}

	return managedKafkas, readyKafka, msg, nil
}

func CountKafkaTopics(ctx context.Context, pClient client.Client, statusSource statusTypes.StatusSource, namespaces []string) (int32, int32, string, error) {
	var managedTopics int32
	var readyTopics int32
	var brokenTopics []string
	var msg = ""

	topics := []strimzi.KafkaTopic{}
	for _, namespace := range namespaces {
		opts := []client.ListOption{
			client.InNamespace(namespace),
		}
		tmpTopics := strimzi.KafkaTopicList{}
		err := pClient.List(ctx, &tmpTopics, opts...)
		if err != nil {
			return 0, 0, "", err
		}
		topics = append(topics, tmpTopics.Items...)
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, kafkaTopic := range topics {
		for _, owner := range kafkaTopic.GetOwnerReferences() {
			if owner.UID == statusSource.GetUID() {
				managedTopics++
				if ok := kafkaTopicStatusChecker(kafkaTopic); ok {
					readyTopics++
				} else {
					brokenTopics = append(brokenTopics, fmt.Sprintf("%s/%s", kafkaTopic.Name, kafkaTopic.Namespace))
				}
				break
			}
		}
	}

	if len(brokenTopics) > 0 {
		sort.Strings(brokenTopics)
		msg = fmt.Sprintf("broken topics: [%s]", strings.Join(brokenTopics, ", "))
	}

	return managedTopics, readyTopics, msg, nil
}

func CountKafkaConnects(ctx context.Context, pClient client.Client, statusSource statusTypes.StatusSource, namespaces []string) (int32, int32, string, error) {
	var managedConnects int32
	var readyConnects int32
	var brokenConnects []string
	var msg = ""

	connects := []strimzi.KafkaConnect{}
	for _, namespace := range namespaces {
		opts := []client.ListOption{
			client.InNamespace(namespace),
		}
		tmpConnects := strimzi.KafkaConnectList{}
		err := pClient.List(ctx, &tmpConnects, opts...)
		if err != nil {
			return 0, 0, "", err
		}
		connects = append(connects, tmpConnects.Items...)
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, kafkaConnect := range connects {
		for _, owner := range kafkaConnect.GetOwnerReferences() {
			if owner.UID == statusSource.GetUID() {
				managedConnects++
				if ok := kafkaConnectStatusChecker(kafkaConnect); ok {
					readyConnects++
				} else {
					brokenConnects = append(brokenConnects, fmt.Sprintf("%s/%s", kafkaConnect.Name, kafkaConnect.Namespace))
				}
				break
			}
		}
	}

	if len(brokenConnects) > 0 {
		msg = fmt.Sprintf("broken connects: [%s]", strings.Join(brokenConnects, ", "))
	}

	return managedConnects, readyConnects, msg, nil
}
