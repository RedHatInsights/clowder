package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	cond "github.com/RedHatInsights/rhc-osdk-utils/conditionhandler"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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

func countDeployments(ctx context.Context, client client.Client, o object.ClowdObject) (int32, int32, error) {
	var managedDeployments int32
	var readyDeployments int32

	deployments := apps.DeploymentList{}
	err := client.List(ctx, &deployments)
	if err != nil {
		return 0, 0, err
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

	return managedDeployments, readyDeployments, nil
}

func countKafkas(ctx context.Context, client client.Client, o object.ClowdObject) (int32, int32, error) {
	var managedDeployments int32
	var readyDeployments int32

	kafkas := strimzi.KafkaList{}
	err := client.List(ctx, &kafkas)
	if err != nil {
		return 0, 0, err
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

	return managedDeployments, readyDeployments, nil
}

func countKafkaTopics(ctx context.Context, client client.Client, o object.ClowdObject) (int32, int32, error) {
	var managedTopics int32
	var readyTopics int32

	kafkaTopics := strimzi.KafkaTopicList{}
	err := client.List(ctx, &kafkaTopics)
	if err != nil {
		return 0, 0, err
	}

	// filter for resources owned by the ClowdObject and check their status
	for _, kafkaTopic := range kafkaTopics.Items {
		for _, owner := range kafkaTopic.GetOwnerReferences() {
			if owner.UID == o.GetUID() {
				managedTopics++
				if ok := kafkaTopicStatusChecker(kafkaTopic); ok {
					readyTopics++
				}
				break
			}
		}
	}

	return managedTopics, readyTopics, nil
}

func countKafkaConnects(ctx context.Context, client client.Client, o object.ClowdObject) (int32, int32, error) {
	var managedDeployments int32
	var readyDeployments int32

	kafkaConnects := strimzi.KafkaConnectList{}
	err := client.List(ctx, &kafkaConnects)
	if err != nil {
		return 0, 0, err
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

	return managedDeployments, readyDeployments, nil
}

// SetEnvResourceStatus the status on the passed ClowdObject interface.
func SetEnvResourceStatus(ctx context.Context, client client.Client, o *crd.ClowdEnvironment) error {
	stats, err := GetEnvResourceFigures(ctx, client, o)
	if err != nil {
		return err
	}

	status := o.GetDeploymentStatus()
	status.ManagedDeployments = stats.ManagedDeployments
	status.ReadyDeployments = stats.ReadyDeployments
	status.ManagedTopics = stats.ManagedTopics
	status.ReadyTopics = stats.ReadyTopics

	return nil
}

func GetEnvResourceFigures(ctx context.Context, client client.Client, o *crd.ClowdEnvironment) (crd.EnvResourceStatus, error) {

	var totalManagedDeployments int32
	var totalReadyDeployments int32
	var totalManagedTopics int32
	var totalReadyTopics int32

	deploymentStats := crd.EnvResourceStatus{}

	managedDeployments, readyDeployments, err := countDeployments(ctx, client, o)
	if err != nil {
		return crd.EnvResourceStatus{}, err
	}
	totalManagedDeployments += managedDeployments
	totalReadyDeployments += readyDeployments

	if clowderconfig.LoadedConfig.Features.WatchStrimziResources {
		managedDeployments, readyDeployments, err = countKafkas(ctx, client, o)
		if err != nil {
			return crd.EnvResourceStatus{}, err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments

		managedDeployments, readyDeployments, err = countKafkaConnects(ctx, client, o)
		if err != nil {
			return crd.EnvResourceStatus{}, err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments

		managedTopics, readyTopics, err := countKafkaTopics(ctx, client, o)
		if err != nil {
			return crd.EnvResourceStatus{}, err
		}
		totalManagedTopics += managedTopics
		totalReadyTopics += readyTopics

	}

	deploymentStats.ManagedDeployments = totalManagedDeployments
	deploymentStats.ReadyDeployments = totalReadyDeployments
	deploymentStats.ManagedTopics = totalManagedTopics
	deploymentStats.ReadyTopics = totalReadyTopics
	return deploymentStats, nil
}

func GetAppResourceStatus(ctx context.Context, client client.Client, o *crd.ClowdApp) (bool, error) {
	stats, err := GetAppResourceFigures(ctx, client, o)
	if err != nil {
		return false, err
	}
	if stats.ManagedDeployments == stats.ReadyDeployments {
		return true, nil
	}
	return false, nil
}

// SetAppResourceStatus the status on the passed ClowdObject interface.
func SetAppResourceStatus(ctx context.Context, client client.Client, o *crd.ClowdApp) error {
	stats, err := GetAppResourceFigures(ctx, client, o)
	if err != nil {
		return err
	}

	status := o.GetDeploymentStatus()
	status.ManagedDeployments = stats.ManagedDeployments
	status.ReadyDeployments = stats.ReadyDeployments

	return nil
}

func GetAppResourceFigures(ctx context.Context, client client.Client, o *crd.ClowdApp) (crd.AppResourceStatus, error) {

	var totalManagedDeployments int32
	var totalReadyDeployments int32

	deploymentStats := crd.AppResourceStatus{}

	managedDeployments, readyDeployments, err := countDeployments(ctx, client, o)
	if err != nil {
		return crd.AppResourceStatus{}, err
	}
	totalManagedDeployments += managedDeployments
	totalReadyDeployments += readyDeployments

	if clowderconfig.LoadedConfig.Features.WatchStrimziResources {
		managedDeployments, readyDeployments, err = countKafkas(ctx, client, o)
		if err != nil {
			return crd.AppResourceStatus{}, err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments

		managedDeployments, readyDeployments, err = countKafkaConnects(ctx, client, o)
		if err != nil {
			return crd.AppResourceStatus{}, err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments

	}

	deploymentStats.ManagedDeployments = totalManagedDeployments
	deploymentStats.ReadyDeployments = totalReadyDeployments
	return deploymentStats, nil
}

func GetEnvResourceStatus(ctx context.Context, client client.Client, o *crd.ClowdEnvironment) (bool, error) {
	stats, err := GetEnvResourceFigures(ctx, client, o)
	if err != nil {
		return false, err
	}
	if stats.ManagedDeployments == stats.ReadyDeployments && stats.ManagedTopics == stats.ReadyTopics {
		return true, nil
	}
	return false, nil
}

func SetClowdEnvConditions(ctx context.Context, client client.Client, o *crd.ClowdEnvironment, state string, err error) error {
	conditions := []v1.Condition{}

	loopConditions := []string{crd.ReconciliationSuccessful, crd.ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &v1.Condition{}
		condition.Type = conditionType
		condition.Status = v1.ConditionFalse

		if state == conditionType {
			condition.Status = v1.ConditionTrue
			if err != nil {
				condition.Reason = err.Error()
			}
		}

		condition.LastTransitionTime = v1.Now()
		conditions = append(conditions, *condition)
	}

	deploymentStatus, err := GetEnvResourceStatus(ctx, client, o)
	if err != nil {
		return err
	}

	condition := &v1.Condition{}

	condition.Status = v1.ConditionFalse
	condition.Message = "Deployments are not yet ready"
	if deploymentStatus {
		condition.Status = v1.ConditionTrue
		condition.Message = "All managed deployments ready"
	}

	condition.Type = crd.DeploymentsReady
	condition.LastTransitionTime = v1.Now()
	if err != nil {
		condition.Reason = err.Error()
	}

	conditions = append(conditions, *condition)

	for _, condition := range conditions {
		cond.UpdateCondition(&o.Status.Conditions, &condition)
	}

	o.Status.Ready = deploymentStatus

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	return nil
}

func SetClowdAppConditions(ctx context.Context, client client.Client, o *crd.ClowdApp, state string, err error) error {
	conditions := []v1.Condition{}

	loopConditions := []string{crd.ReconciliationSuccessful, crd.ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &v1.Condition{}
		condition.Type = conditionType
		condition.Status = v1.ConditionFalse

		if state == conditionType {
			condition.Status = v1.ConditionTrue
			if err != nil {
				condition.Reason = err.Error()
			}
		}

		condition.LastTransitionTime = v1.Now()
		conditions = append(conditions, *condition)
	}

	deploymentStatus, err := GetAppResourceStatus(ctx, client, o)
	if err != nil {
		return err
	}

	condition := &v1.Condition{}

	condition.Status = v1.ConditionFalse
	condition.Message = "Deployments are not yet ready"
	if deploymentStatus {
		condition.Status = v1.ConditionTrue
		condition.Message = "All managed deployments ready"
	}

	condition.Type = crd.DeploymentsReady
	condition.LastTransitionTime = v1.Now()
	if err != nil {
		condition.Reason = err.Error()
	}

	conditions = append(conditions, *condition)

	for _, condition := range conditions {
		cond.UpdateCondition(&o.Status.Conditions, &condition)
	}

	o.Status.Ready = deploymentStatus

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	return nil
}

func SetClowdJobInvocationConditions(ctx context.Context, client client.Client, o *crd.ClowdJobInvocation, state string, err error) error {
	conditions := []v1.Condition{}

	loopConditions := []string{crd.ReconciliationSuccessful, crd.ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &v1.Condition{}
		condition.Type = conditionType
		condition.Status = v1.ConditionFalse

		if state == conditionType {
			condition.Status = v1.ConditionTrue
			if err != nil {
				condition.Reason = err.Error()
			}
		}

		condition.LastTransitionTime = v1.Now()
		conditions = append(conditions, *condition)
	}

	// Setup custom status for CJI
	condition := &v1.Condition{}
	condition.Type = crd.JobInvocationComplete
	condition.Status = v1.ConditionFalse
	condition.Message = "Some Jobs are still incomplete"
	condition.LastTransitionTime = v1.Now()
	if err != nil {
		condition.Reason = err.Error()
	}

	jobs, err := o.GetInvokedJobs(ctx, client)
	if err != nil {
		return err
	}
	jobStatus := GetJobsStatus(jobs, o)

	if jobStatus {
		condition.Status = v1.ConditionTrue
		condition.Message = "All ClowdJob invocations complete"
	}
	conditions = append(conditions, *condition)

	for _, condition := range conditions {
		cond.UpdateCondition(&o.Status.Conditions, &condition)
	}

	o.Status.Completed = jobStatus
	UpdateInvokedJobStatus(ctx, jobs, o)

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1464#issuecomment-811930090
	// Handle the lag between the client and the k8s cache
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
