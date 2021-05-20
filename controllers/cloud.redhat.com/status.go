package controllers

import (
	"context"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func filterOwnedDeployments(deploymentList *apps.DeploymentList, uid types.UID) {
	depList := []apps.Deployment{}
	for _, deployment := range deploymentList.Items {
		for _, owner := range deployment.ObjectMeta.OwnerReferences {
			if owner.UID == uid {
				depList = append(depList, deployment)
			}
		}
	}
	deploymentList.Items = depList
}

// SetDeploymentStatus the status on the passed ClowdObject interface.
func SetDeploymentStatus(ctx context.Context, client client.Client, o object.ClowdObject) error {
	deploymentList := apps.DeploymentList{}
	err := client.List(ctx, &deploymentList)

	if err != nil {
		return err
	}

	filterOwnedDeployments(&deploymentList, o.GetUID())
	var managedDeployments int32
	var readyDeployments int32

	for _, deployment := range deploymentList.Items {
		managedDeployments++
		if ok := utils.DeploymentStatusChecker(&deployment); ok {
			readyDeployments++
		}
	}

	status := o.GetDeploymentStatus()
	status.ManagedDeployments = managedDeployments
	status.ReadyDeployments = readyDeployments

	return nil
}
