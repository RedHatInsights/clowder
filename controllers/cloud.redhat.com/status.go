package controllers

import (
	"context"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func filterOwnedObjects(objectList *client.ObjectList, uid types.UID) {
	filteredObjects := []client.ObjectList{}
	for _, obj := range objectList.Items {
		for _, owner := range obj.ObjectMeta.OwnerReferences {
			if owner.UID == uid {
				filteredObjects = append(filteredObjects, obj)
			}
		}
	}
	objectList.Items = filteredObjects
}

func getDeploymentStatus(ctx context.Context, client client.Client, o object.ClowdObject) (error, int32, int32) {
	objectList := client.ObjectList{}
	err := client.List(ctx, &objectList)

	if err != nil {
		return err, 0, 0
	}

	filterOwnedObjects(objectList, o.GetUID())
	var managedDeployments int32
	var readyDeployments int32

	for _, obj := range objectList.Items {
		switch t := obj.(type) {
		case apps.Deployment:
			managedDeployments++
			if ok := utils.DeploymentStatusChecker(&obj.(apps.Deployment)); ok {
				readyDeployments++
			}
		case strimzi.Kafka:
			managedDeployments++
			// TODO: actually check for ready
			readyDeployments++
		case strimzi.KafkaConnect:
			managedDeployments++
			// TODO: actually check for ready
			readyDeployments++
		}

	}

	return nil, managedDeployments, readyDeployments
}

// SetDeploymentStatus the status on the passed ClowdObject interface.
func SetDeploymentStatus(ctx context.Context, client client.Client, o object.ClowdObject) error {
	err, managedDeployments, readyDeployments := getDeploymentStatus(ctx, client, o)
	if err != nil {
		return err
	}

	status := o.GetDeploymentStatus()
	status.ManagedDeployments = managedDeployments
	status.ReadyDeployments = readyDeployments

	return nil
}
