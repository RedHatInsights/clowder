package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/rhc-osdk-utils/resources"
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

func countDeployments(ctx context.Context, pClient client.Client, uid types.UID, namespaces []string) resources.ResourceCounterResults {
	query, _ := resources.MakeQuery(&apps.Deployment{}, *pClient.Scheme(), namespaces, uid)
	counter := resources.ResourceCounter{
		Query: query,
		ReadyRequirements: []resources.ResourceConditionReadyRequirements{{
			Type:   "Available",
			Status: "True",
		}},
	}
	results := counter.Count(ctx, pClient)
	return results
}

func countKafkas(ctx context.Context, pClient client.Client, uid types.UID, namespaces []string) resources.ResourceCounterResults {
	query, _ := resources.MakeQuery(&strimzi.Kafka{}, *pClient.Scheme(), namespaces, uid)
	counter := resources.ResourceCounter{
		Query: query,
		ReadyRequirements: []resources.ResourceConditionReadyRequirements{{
			Type:   "Ready",
			Status: "True",
		}},
	}
	results := counter.Count(ctx, pClient)
	return results
}

func countKafkaConnects(ctx context.Context, pClient client.Client, uid types.UID, namespaces []string) resources.ResourceCounterResults {
	query, _ := resources.MakeQuery(&strimzi.KafkaConnect{}, *pClient.Scheme(), namespaces, uid)
	counter := resources.ResourceCounter{
		Query: query,
		ReadyRequirements: []resources.ResourceConditionReadyRequirements{{
			Type:   "Ready",
			Status: "True",
		}},
	}
	results := counter.Count(ctx, pClient)
	return results
}

func countKafkaTopics(ctx context.Context, pClient client.Client, uid types.UID, namespaces []string) resources.ResourceCounterResults {
	query, _ := resources.MakeQuery(&strimzi.KafkaTopic{}, *pClient.Scheme(), namespaces, uid)
	counter := resources.ResourceCounter{
		Query: query,
		ReadyRequirements: []resources.ResourceConditionReadyRequirements{{
			Type:   "Ready",
			Status: "True",
		}},
	}
	results := counter.Count(ctx, pClient)
	return results
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

type checkFn func(ctx context.Context, pClient client.Client, uid types.UID, namespaces []string) resources.ResourceCounterResults
type statusChecking struct {
	fn      checkFn
	managed *int32
	ready   *int32
}

func GetEnvResourceFigures(ctx context.Context, client client.Client, o *crd.ClowdEnvironment) (crd.EnvResourceStatus, string, error) {
	var msgs []string

	deploymentStats := crd.EnvResourceStatus{}

	namespaces, err := o.GetNamespacesInEnv(ctx, client)
	if err != nil {
		return crd.EnvResourceStatus{}, "", err
	}

	uid := o.GetUID()

	checkFns := []statusChecking{{
		fn:      countDeployments,
		managed: &deploymentStats.ManagedDeployments,
		ready:   &deploymentStats.ReadyDeployments,
	}}
	if clowderconfig.LoadedConfig.Features.WatchStrimziResources {
		kafkaFns := []statusChecking{
			{
				fn:      countKafkas,
				managed: &deploymentStats.ManagedDeployments,
				ready:   &deploymentStats.ReadyDeployments,
			},
			{
				fn:      countKafkaConnects,
				managed: &deploymentStats.ManagedDeployments,
				ready:   &deploymentStats.ReadyDeployments,
			},
			{
				fn:      countKafkaTopics,
				managed: &deploymentStats.ManagedTopics,
				ready:   &deploymentStats.ReadyTopics,
			},
		}
		checkFns = append(checkFns, kafkaFns...)
	}

	for _, fn := range checkFns {
		results := fn.fn(ctx, client, uid, namespaces)
		*fn.managed += int32(results.Managed)
		*fn.ready += int32(results.Ready)
		if results.BrokenMessage != "" {
			msgs = append(msgs, results.BrokenMessage)
		}
	}

	msg := fmt.Sprintf("dependency failure: [%s]", strings.Join(msgs, ","))

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

func GetAppResourceFigures(ctx context.Context, client client.Client, o object.ClowdObject) (crd.AppResourceStatus, string, error) {
	deploymentStats := crd.AppResourceStatus{}

	namespaces, err := o.GetNamespacesInEnv(ctx, client)
	if err != nil {
		return crd.AppResourceStatus{}, "", errors.Wrap("get namespaces: ", err)
	}

	results := countDeployments(ctx, client, o.GetUID(), namespaces)

	deploymentStats.ManagedDeployments = int32(results.Managed)
	deploymentStats.ReadyDeployments = int32(results.Ready)
	return deploymentStats, results.BrokenMessage, nil
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
