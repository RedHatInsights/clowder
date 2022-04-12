package status

import (
	"context"
	"fmt"
	"sort"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	cond "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	statusTypes "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/status/types"
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

func countDeployments(ctx context.Context, pClient client.Client, statusSource statusTypes.StatusSource, namespaces []string) (int32, int32, string, error) {
	var managedDeployments int32
	var readyDeployments int32
	var brokenDeployments []string
	var msg = ""

	deployments := []apps.Deployment{}
	for _, namespace := range namespaces {
		opts := []client.ListOption{
			client.InNamespace(namespace),
		}
		tmpDeployments := apps.DeploymentList{}
		err := pClient.List(ctx, &tmpDeployments, opts...)
		if err != nil {
			return 0, 0, "", err
		}
		deployments = append(deployments, tmpDeployments.Items...)
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, deployment := range deployments {
		for _, owner := range deployment.GetOwnerReferences() {
			if owner.UID == statusSource.GetUID() {
				managedDeployments++
				if ok := deploymentStatusChecker(deployment); ok {
					readyDeployments++
				} else {
					brokenDeployments = append(brokenDeployments, fmt.Sprintf("%s/%s", deployment.Name, deployment.Namespace))
				}
				break
			}
		}
	}

	if len(brokenDeployments) > 0 {
		sort.Strings(brokenDeployments)
		msg = fmt.Sprintf("broken deployments: [%s]", strings.Join(brokenDeployments, ", "))
	}

	return managedDeployments, readyDeployments, msg, nil
}

func SetResourceStatus(ctx context.Context, client client.Client, statusSource statusTypes.StatusSource) error {
	stats, _, err := GetResourceFigures(ctx, client, statusSource)
	if err != nil {
		return err
	}

	statusSource.SetDeploymentFigures(stats)

	return nil
}

func GetResourceFigures(ctx context.Context, client client.Client, statusSource statusTypes.StatusSource) (statusTypes.StatusSourceFigures, string, error) {
	figures := statusTypes.StatusSourceFigures{}
	msg := ""
	namespaces, err := statusSource.GetNamespaces(ctx, client)
	if err != nil {
		return figures, "", errors.Wrap("get namespaces: ", err)
	}

	managedDeployments, readyDeployments, _, err := countDeployments(ctx, client, statusSource, namespaces)
	if err != nil {
		return figures, "", errors.Wrap("count deploys: ", err)
	}

	figures.ManagedDeployments += managedDeployments
	figures.ReadyDeployments += readyDeployments

	specialFigures, msg, err := statusSource.GetObjectSpecificFigures(ctx, client)
	if err != nil {
		return figures, msg, err
	}
	figures = statusSource.AddDeploymentFigures(figures, specialFigures)

	return figures, msg, nil
}

func GetResourceStatus(ctx context.Context, client client.Client, statusSource statusTypes.StatusSource) (bool, string, error) {
	stats, msg, err := GetResourceFigures(ctx, client, statusSource)
	if err != nil {
		return false, msg, err
	}
	return statusSource.AreDeploymentsReady(stats), msg, nil
}

func SetConditions(ctx context.Context, client client.Client, statusSource statusTypes.StatusSource, state clusterv1.ConditionType, err error) error {
	conditions := []clusterv1.Condition{}

	loopConditions := []clusterv1.ConditionType{crd.ReconciliationSuccessful, crd.ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &clusterv1.Condition{}
		condition.Type = conditionType
		condition.Status = core.ConditionFalse

		if state == conditionType {
			condition.Status = core.ConditionTrue
			if err != nil {
				condition.Reason = err.Error()
			}
		}

		condition.LastTransitionTime = v1.Now()
		conditions = append(conditions, *condition)
	}

	deploymentStatus, _, err := GetResourceStatus(ctx, client, statusSource)
	if err != nil {
		return err
	}

	condition := &clusterv1.Condition{}

	condition.Status = core.ConditionFalse
	condition.Message = "Deployments are not yet ready"
	if deploymentStatus {
		condition.Status = core.ConditionTrue
		condition.Message = "All managed deployments ready"
	}

	condition.Type = crd.DeploymentsReady
	condition.LastTransitionTime = v1.Now()
	if err != nil {
		condition.Reason = err.Error()
	}

	conditions = append(conditions, *condition)

	for _, condition := range conditions {
		cond.Set(statusSource, &condition)
	}

	statusSource.SetStatusReady(deploymentStatus)

	if err := client.Status().Update(ctx, statusSource); err != nil {
		return err
	}
	return nil
}
