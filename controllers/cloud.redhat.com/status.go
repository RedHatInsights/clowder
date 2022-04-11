package controllers

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	cond "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func deploymentStatusChecker(deployment apps.Deployment) bool {
	s := StatusProcessor{}
	s.ProcessDeployment(deployment)
	return s.GetStatus()
}

func kafkaStatusChecker(kafka strimzi.Kafka) bool {
	s := StatusProcessor{}
	s.ProcessKafka(kafka)
	return s.GetStatus()
}

func kafkaTopicStatusChecker(kafkaTopic strimzi.KafkaTopic) bool {
	s := StatusProcessor{}
	s.ProcessKafkaTopic(kafkaTopic)
	return s.GetStatus()
}

func kafkaConnectStatusChecker(kafkaConnect strimzi.KafkaConnect) bool {
	s := StatusProcessor{}
	s.ProcessKafkaConnect(kafkaConnect)
	return s.GetStatus()
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

func SetClowdEnvConditions(ctx context.Context, client client.Client, o *crd.ClowdEnvironment, state clusterv1.ConditionType, err error) error {
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
		cond.Set(o, &condition)
	}

	o.Status.Ready = deploymentStatus

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	return nil
}

//Top level function called by reconciler
//Called when something is reconciled or when reconciliation fails
func SetClowdAppConditions(ctx context.Context, client client.Client, o *crd.ClowdApp, state clusterv1.ConditionType, err error) error {
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
		cond.Set(o, &condition)
	}

	o.Status.Ready = deploymentStatus

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	return nil
}

func SetClowdJobInvocationConditions(ctx context.Context, client client.Client, o *crd.ClowdJobInvocation, state clusterv1.ConditionType, err error) error {
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
		cond.Set(o, &condition)
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
