package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowder_config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentStats struct {
	ManagedDeployments int32
	ReadyDeployments   int32
}

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
	stats, err := GetDeploymentFigures(ctx, client, o)
	if err != nil {
		return err
	}

	status := o.GetDeploymentStatus()
	status.ManagedDeployments = stats.ManagedDeployments
	status.ReadyDeployments = stats.ReadyDeployments

	return nil
}

func GetDeploymentFigures(ctx context.Context, client client.Client, o object.ClowdObject) (DeploymentStats, error) {

	var totalManagedDeployments int32
	var totalReadyDeployments int32

	deploymentStats := DeploymentStats{}

	err, managedDeployments, readyDeployments := countDeployments(ctx, client, o)
	if err != nil {
		return DeploymentStats{}, err
	}
	totalManagedDeployments += managedDeployments
	totalReadyDeployments += readyDeployments

	if clowder_config.LoadedConfig.Features.WatchStrimziResources {
		err, managedDeployments, readyDeployments = countKafkas(ctx, client, o)
		if err != nil {
			return DeploymentStats{}, err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments

		err, managedDeployments, readyDeployments = countKafkaConnects(ctx, client, o)
		if err != nil {
			return DeploymentStats{}, err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments
	}

	deploymentStats.ManagedDeployments = totalManagedDeployments
	deploymentStats.ReadyDeployments = totalReadyDeployments
	return deploymentStats, nil
}

func GetDeploymentStatus(ctx context.Context, client client.Client, o object.ClowdObject) (bool, error) {
	stats, err := GetDeploymentFigures(ctx, client, o)
	if err != nil {
		return false, err
	}
	if stats.ManagedDeployments == stats.ReadyDeployments {
		return true, nil
	}
	return false, nil
}

func SetClowdEnvConditions(ctx context.Context, client client.Client, o *crd.ClowdEnvironment, state crd.ClowdConditionType, err error) error {
	conditions := []crd.ClowdCondition{}

	loopConditions := []crd.ClowdConditionType{crd.ReconciliationSuccessful, crd.ReconciliationPartiallySuccessful, crd.ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &crd.ClowdCondition{}
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

	deploymentStatus, err := GetDeploymentStatus(ctx, client, o)
	if err != nil {
		return err
	}

	condition := &crd.ClowdCondition{}

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
		UpdateClowdEnvCondition(&o.Status, &condition)
	}

	o.Status.Ready = deploymentStatus

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	return nil
}

func SetClowdAppConditions(ctx context.Context, client client.Client, o *crd.ClowdApp, state crd.ClowdConditionType, err error) error {
	conditions := []crd.ClowdCondition{}

	loopConditions := []crd.ClowdConditionType{crd.ReconciliationSuccessful, crd.ReconciliationPartiallySuccessful, crd.ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &crd.ClowdCondition{}
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

	deploymentStatus, err := GetDeploymentStatus(ctx, client, o)
	if err != nil {
		return err
	}

	condition := &crd.ClowdCondition{}

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
		UpdateClowdAppCondition(&o.Status, &condition)
	}

	o.Status.Ready = deploymentStatus

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	return nil
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func GetClowdAppConditionFromList(conditions []crd.ClowdCondition, conditionType crd.ClowdConditionType) (int, *crd.ClowdCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func GetClowdAppCondition(status *crd.ClowdAppStatus, conditionType crd.ClowdConditionType) (int, *crd.ClowdCondition) {
	if status == nil {
		return -1, nil
	}
	return GetClowdAppConditionFromList(status.Conditions, conditionType)
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func UpdateClowdAppCondition(status *crd.ClowdAppStatus, condition *crd.ClowdCondition) bool {
	condition.LastTransitionTime = v1.Now()
	// Try to find this clowdapp condition.
	conditionIndex, oldCondition := GetClowdAppCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new pod condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func GetClowdEnvConditionFromList(conditions []crd.ClowdCondition, conditionType crd.ClowdConditionType) (int, *crd.ClowdCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func GetClowdEnvCondition(status *crd.ClowdEnvironmentStatus, conditionType crd.ClowdConditionType) (int, *crd.ClowdCondition) {
	if status == nil {
		return -1, nil
	}
	return GetClowdEnvConditionFromList(status.Conditions, conditionType)
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func UpdateClowdEnvCondition(status *crd.ClowdEnvironmentStatus, condition *crd.ClowdCondition) bool {
	condition.LastTransitionTime = v1.Now()
	// Try to find this clowdapp condition.
	conditionIndex, oldCondition := GetClowdEnvCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new pod condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func GetClowdJobInvocationConditionFromList(conditions []crd.ClowdCondition, conditionType crd.ClowdConditionType) (int, *crd.ClowdCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func GetClowdJobInvocationCondition(status *crd.ClowdJobInvocationStatus, conditionType crd.ClowdConditionType) (int, *crd.ClowdCondition) {
	if status == nil {
		return -1, nil
	}
	return GetClowdJobInvocationConditionFromList(status.Conditions, conditionType)
}

// The following function was modified from the kubnernetes repo under the apache license here
// https://github.com/kubernetes/kubernetes/blob/v1.21.1/pkg/api/v1/pod/util.go#L317-L367
func UpdateClowdJobInvocationCondition(status *crd.ClowdJobInvocationStatus, condition *crd.ClowdCondition) bool {
	condition.LastTransitionTime = v1.Now()
	conditionIndex, oldCondition := GetClowdJobInvocationCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new pod condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

func SetClowdJobInvocationConditions(ctx context.Context, client client.Client, o *crd.ClowdJobInvocation, state crd.ClowdConditionType, err error) error {
	conditions := []crd.ClowdCondition{}

	loopConditions := []crd.ClowdConditionType{crd.ReconciliationSuccessful, crd.ReconciliationPartiallySuccessful, crd.ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &crd.ClowdCondition{}
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

	jobs, err := o.GetInvokedJobs(ctx, client)
	if err != nil {
		return err
	}
	jobStatus := GetJobsStatus(jobs, o)

	condition := &crd.ClowdCondition{}

	condition.Status = core.ConditionFalse
	condition.Message = "Some Jobs are still incomplete"

	if jobStatus {
		condition.Status = core.ConditionTrue
		condition.Message = "All ClowdJob invocations complete"
	}

	condition.Type = crd.JobInvocationComplete
	condition.LastTransitionTime = v1.Now()
	if err != nil {
		condition.Reason = err.Error()
	}

	conditions = append(conditions, *condition)

	for _, condition := range conditions {
		UpdateClowdJobInvocationCondition(&o.Status, &condition)
	}

	o.Status.Completed = jobStatus
	UpdateInvokedJobStatus(ctx, jobs, o)

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	cjiState := o.Status
	nn := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}
	if err := wait.Poll(100*time.Millisecond, 2*time.Second, func() (bool, error) {
		if err := client.Get(ctx, nn, o); err != nil {
			return false, fmt.Errorf("failed to get cji: %w", err)
		}
		return reflect.DeepEqual(o.Status, cjiState), nil
	}); err != nil {
		return fmt.Errorf("failed to wait for cached cji %s to get into state %s: %w", nn.String(), state, err)
	}
	return nil
}
