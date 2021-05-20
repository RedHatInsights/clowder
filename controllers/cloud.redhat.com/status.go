package controllers

import (
	"context"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/clowder_config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta1"
	apps "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func deploymentStatusChecker(deployment apps.Deployment) bool {
	if deployment.Generation > deployment.Status.ObservedGeneration {
		// The status on this resource needs to update
		return false
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == "Available" && condition.Status == "True" {
			return true
		}
	}

	return false
}

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

func countDeployments(ctx context.Context, client client.Client, o object.ClowdObject) (error, int32, int32) {
	var managedDeployments int32
	var readyDeployments int32

	deployments := apps.DeploymentList{}
	err := client.List(ctx, &deployments)
	if err != nil {
		return err, 0, 0
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, deployment := range deployments.Items {
		for _, owner := range deployment.GetOwnerReferences() {
			if owner.UID == o.GetUID() {
				managedDeployments++
				if ok := deploymentStatusChecker(deployment); ok {
					readyDeployments++
				}
				break
			}
		}
	}

	return nil, managedDeployments, readyDeployments
}

func countKafkas(ctx context.Context, client client.Client, o object.ClowdObject) (error, int32, int32) {
	var managedDeployments int32
	var readyDeployments int32

	kafkas := strimzi.KafkaList{}
	err := client.List(ctx, &kafkas)
	if err != nil {
		return err, 0, 0
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, kafka := range kafkas.Items {
		for _, owner := range kafka.GetOwnerReferences() {
			if owner.UID == o.GetUID() {
				managedDeployments++
				if ok := kafkaStatusChecker(kafka); ok {
					readyDeployments++
				}
				break
			}
		}
	}

	return nil, managedDeployments, readyDeployments
}

func countKafkaConnects(ctx context.Context, client client.Client, o object.ClowdObject) (error, int32, int32) {
	var managedDeployments int32
	var readyDeployments int32

	kafkaConnects := strimzi.KafkaConnectList{}
	err := client.List(ctx, &kafkaConnects)
	if err != nil {
		return err, 0, 0
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, kafkaConnect := range kafkaConnects.Items {
		for _, owner := range kafkaConnect.GetOwnerReferences() {
			if owner.UID == o.GetUID() {
				managedDeployments++
				if ok := kafkaConnectStatusChecker(kafkaConnect); ok {
					readyDeployments++
				}
				break
			}
		}
	}

	return nil, managedDeployments, readyDeployments
}

// SetDeploymentStatus the status on the passed ClowdObject interface.
func SetDeploymentStatus(ctx context.Context, client client.Client, o object.ClowdObject) error {
	var totalManagedDeployments int32
	var totalReadyDeployments int32

	err, managedDeployments, readyDeployments := countDeployments(ctx, client, o)
	if err != nil {
		return err
	}
	totalManagedDeployments += managedDeployments
	totalReadyDeployments += readyDeployments

	if clowder_config.LoadedConfig.Features.WatchStrimziResources {
		err, managedDeployments, readyDeployments = countKafkas(ctx, client, o)
		if err != nil {
			return err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments

		err, managedDeployments, readyDeployments = countKafkaConnects(ctx, client, o)
		if err != nil {
			return err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments
	}

	status := o.GetDeploymentStatus()
	status.ManagedDeployments = totalManagedDeployments
	status.ReadyDeployments = totalReadyDeployments

	return nil
}
