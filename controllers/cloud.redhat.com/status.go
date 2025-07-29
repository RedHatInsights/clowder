package controllers

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	cond "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
)

func deploymentStatusChecker(deployment apps.Deployment) bool {
	if deployment.Generation > deployment.Status.ObservedGeneration {
		// The status on this resource needs to update
		return false
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == "Available" &&
			condition.Status == "True" &&
			deployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
			deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas {

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

func countDeployments(ctx context.Context, pClient client.Client, o object.ClowdObject, namespaces []string) (int32, int32, string, error) {
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
			if owner.UID == o.GetUID() {
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

func countKafkas(ctx context.Context, pClient client.Client, o object.ClowdObject, namespaces []string) (int32, int32, string, error) {
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
			if owner.UID == o.GetUID() {
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

func countKafkaTopics(ctx context.Context, pClient client.Client, o object.ClowdObject, namespaces []string) (int32, int32, string, error) {
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
			if owner.UID == o.GetUID() {
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

func countKafkaConnects(ctx context.Context, pClient client.Client, o object.ClowdObject, namespaces []string) (int32, int32, string, error) {
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
			if owner.UID == o.GetUID() {
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

// SetEnvResourceStatus the status on the passed ClowdObject interface.
func SetEnvResourceStatus(ctx context.Context, client client.Client, o *crd.ClowdEnvironment) error {
	stats, _, err := GetEnvResourceFigures(ctx, client, o)
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

func GetEnvResourceFigures(ctx context.Context, client client.Client, o *crd.ClowdEnvironment) (crd.EnvResourceStatus, string, error) {

	var totalManagedDeployments int32
	var totalReadyDeployments int32
	var totalManagedTopics int32
	var totalReadyTopics int32
	var msgs []string

	deploymentStats := crd.EnvResourceStatus{}

	namespaces, err := o.GetNamespacesInEnv(ctx, client)
	if err != nil {
		return crd.EnvResourceStatus{}, "", err
	}

	managedDeployments, readyDeployments, msg, err := countDeployments(ctx, client, o, namespaces)
	if err != nil {
		return crd.EnvResourceStatus{}, "", err
	}
	totalManagedDeployments += managedDeployments
	totalReadyDeployments += readyDeployments
	if msg != "" {
		msgs = append(msgs, msg)
	}

	if clowderconfig.LoadedConfig.Features.WatchStrimziResources {
		managedDeployments, readyDeployments, msg, err = countKafkas(ctx, client, o, namespaces)
		if err != nil {
			return crd.EnvResourceStatus{}, "", err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments
		if msg != "" {
			msgs = append(msgs, msg)
		}

		managedDeployments, readyDeployments, msg, err = countKafkaConnects(ctx, client, o, namespaces)
		if err != nil {
			return crd.EnvResourceStatus{}, "", err
		}
		totalManagedDeployments += managedDeployments
		totalReadyDeployments += readyDeployments
		if msg != "" {
			msgs = append(msgs, msg)
		}

		managedTopics, readyTopics, msg, err := countKafkaTopics(ctx, client, o, namespaces)
		if err != nil {
			return crd.EnvResourceStatus{}, "", err
		}
		totalManagedTopics += managedTopics
		totalReadyTopics += readyTopics
		if msg != "" {
			msgs = append(msgs, msg)
		}
	}

	msg = fmt.Sprintf("dependency failure: [%s]", strings.Join(msgs, ","))
	deploymentStats.ManagedDeployments = totalManagedDeployments
	deploymentStats.ReadyDeployments = totalReadyDeployments
	deploymentStats.ManagedTopics = totalManagedTopics
	deploymentStats.ReadyTopics = totalReadyTopics
	return deploymentStats, msg, nil
}

func GetAppResourceStatus(ctx context.Context, client client.Client, o *crd.ClowdApp) (bool, error) {
	stats, _, err := GetAppResourceFigures(ctx, client, o)
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
	stats, _, err := GetAppResourceFigures(ctx, client, o)
	if err != nil {
		return err
	}

	status := o.GetDeploymentStatus()
	status.ManagedDeployments = stats.ManagedDeployments
	status.ReadyDeployments = stats.ReadyDeployments

	return nil
}

func GetAppResourceFigures(ctx context.Context, client client.Client, o *crd.ClowdApp) (crd.AppResourceStatus, string, error) {

	var totalManagedDeployments int32
	var totalReadyDeployments int32
	var msgs []string

	deploymentStats := crd.AppResourceStatus{}

	namespaces, err := o.GetNamespacesInEnv(ctx, client)
	if err != nil {
		return crd.AppResourceStatus{}, "", errors.Wrap("get namespaces: ", err)
	}

	managedDeployments, readyDeployments, msg, err := countDeployments(ctx, client, o, namespaces)
	if err != nil {
		return crd.AppResourceStatus{}, "", errors.Wrap("count deploys: ", err)
	}
	totalManagedDeployments += managedDeployments
	totalReadyDeployments += readyDeployments
	if msg != "" {
		msgs = append(msgs, msg)
	}

	msg = fmt.Sprintf("dependency failure: [%s]", strings.Join(msgs, ","))
	deploymentStats.ManagedDeployments = totalManagedDeployments
	deploymentStats.ReadyDeployments = totalReadyDeployments
	return deploymentStats, msg, nil
}

func GetEnvResourceStatus(ctx context.Context, client client.Client, o *crd.ClowdEnvironment) (bool, string, error) {
	stats, msg, err := GetEnvResourceFigures(ctx, client, o)
	if err != nil {
		return false, msg, err
	}
	if stats.ManagedDeployments == stats.ReadyDeployments && stats.ManagedTopics == stats.ReadyTopics {
		return true, msg, nil
	}
	return false, msg, nil
}

func SetClowdEnvConditions(ctx context.Context, client client.Client, o *crd.ClowdEnvironment, state clusterv1.ConditionType, oldStatus *crd.ClowdEnvironmentStatus, err error) error {
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

	deploymentStatus, msg, err := GetEnvResourceStatus(ctx, client, o)
	if err != nil {
		return err
	}

	condition := &clusterv1.Condition{}

	condition.Status = core.ConditionFalse
	condition.Message = fmt.Sprintf("Deployments are not yet ready: %s", msg)
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
		innerCondition := condition
		cond.Set(o, &innerCondition)
	}

	o.Status.Ready = deploymentStatus

	if !equality.Semantic.DeepEqual(*oldStatus, o.Status) {
		if err := client.Status().Update(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

func SetClowdAppConditions(ctx context.Context, client client.Client, o *crd.ClowdApp, state clusterv1.ConditionType, oldStatus *crd.ClowdAppStatus, err error) error {
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

	deploymentStatus, err := GetAppResourceStatus(ctx, client, o)
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
		innerCondition := condition
		cond.Set(o, &innerCondition)
	}

	o.Status.Ready = deploymentStatus

	if !equality.Semantic.DeepEqual(*oldStatus, o.Status) {
		if err := client.Status().Update(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

func SetClowdJobInvocationConditions(ctx context.Context, client client.Client, o *crd.ClowdJobInvocation, state clusterv1.ConditionType, err error) error {
	oldStatus := o.Status.DeepCopy()
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

	// Setup custom status for CJI
	condition := &clusterv1.Condition{}
	condition.Type = crd.JobInvocationComplete
	condition.Status = core.ConditionFalse
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
		condition.Status = core.ConditionTrue
		condition.Message = "All ClowdJob invocations complete"
	}
	conditions = append(conditions, *condition)

	for _, condition := range conditions {
		innerCondition := condition
		cond.Set(o, &innerCondition)
	}

	o.Status.Completed = jobStatus
	// Purposefully clobber this err
	_ = UpdateInvokedJobStatus(jobs, o)

	if !equality.Semantic.DeepEqual(*oldStatus, o.Status) {
		if err := client.Status().Update(ctx, o); err != nil {
			return err
		}
	}
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1464#issuecomment-811930090
	// Handle the lag between the client and the k8s cache
	cjiState := o.Status
	nn := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}
	if err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 2*time.Second, false, func(ctx context.Context) (bool, error) {
		if err := client.Get(ctx, nn, o); err != nil {
			return false, fmt.Errorf("failed to get cji: %w", err)
		}
		return reflect.DeepEqual(o.Status, cjiState), nil
	}); err != nil {
		return fmt.Errorf("failed to wait for cached cji %s to get into state %s: %w", nn.String(), state, err)
	}
	return nil
}
