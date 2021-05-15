package controllers

import (
	"context"
	"fmt"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func filterOwnedObjects(objectList *unstructured.UnstructuredList, uid types.UID) {
	filteredObjects := []unstructured.Unstructured{}
	for _, obj := range objectList.Items {
		for _, owner := range obj.GetOwnerReferences() {
			if owner.UID == uid {
				filteredObjects = append(filteredObjects, obj)
			}
		}
	}
	objectList.Items = filteredObjects
}

func statusConditionPresent(content map[string]interface{}, desiredStatusType string) bool {
	conditions, found, err := unstructured.NestedSlice(content, "status", "conditions")
	if err != nil || !found {
		return false
	}

	for _, condition := range conditions {
		// NestedSlice returns each condition item as an interface{}, we know it should be a map[string]interface{}
		c, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		isStatus, found, err := unstructured.NestedString(c, "status")
		if err != nil || !found || isStatus != "True" {
			continue
		}

		conditionType, found, err := unstructured.NestedString(c, "type")
		if err != nil || !found {
			continue
		}

		if conditionType == desiredStatusType {
			return true
		}
	}

	return false
}

func parseObjects(objectList unstructured.UnstructuredList) (error, int32, int32) {
	var managedDeployments int32
	var readyDeployments int32

	for _, obj := range objectList.Items {
		gvk := obj.GroupVersionKind()
		content := obj.UnstructuredContent()

		if gvk == gvksForType["deployment"].Single {
			// List of deployments
			deployment := apps.Deployment{}
			runtime.DefaultUnstructuredConverter.FromUnstructured(content, &deployment)
			managedDeployments++
			if ok := utils.DeploymentStatusChecker(&deployment); ok {
				readyDeployments++
			}
		} else if gvk == gvksForType["kafka"].Single || gvk == gvksForType["kafkaconnect"].Single {
			// List of Kafka/KafkaConnect resources
			managedDeployments++
			if ok := statusConditionPresent(content, "Ready"); ok {
				readyDeployments++
			}
		} else {
			return fmt.Errorf("unable to parse status for gvk: %s", gvk), 0, 0
		}
	}

	return nil, managedDeployments, readyDeployments
}

// SetDeploymentStatus the status on the passed ClowdObject interface.
func SetDeploymentStatus(ctx context.Context, client client.Client, o object.ClowdObject) error {
	var totalManagedDeployments int32
	var totalReadyDeployments int32

	for _, gvks := range gvksForType {
		objectList := unstructured.UnstructuredList{}
		objectList.SetGroupVersionKind(gvks.List)
		err := client.List(ctx, &objectList)

		if err != nil {
			return err
		}

		filterOwnedObjects(&objectList, o.GetUID())
		err, managedDeployments, readyDeployments := parseObjects(objectList)

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
