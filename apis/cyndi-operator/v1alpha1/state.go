package v1alpha1

import (
	"fmt"
	"strings"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineState describes the state of the pipeline.
type PipelineState string

const (
	// StateNew is the new state of the pipeline.
	StateNew PipelineState = "NEW"

	// StateInitialSync is the new state of the pipeline.
	StateInitialSync PipelineState = "INITIAL_SYNC"

	// StateValid is the new state of the pipeline.
	StateValid PipelineState = "VALID"

	// StateInvalid is the new state of the pipeline.
	StateInvalid PipelineState = "INVALID"

	// StateRemoved is the new state of the pipeline.
	StateRemoved PipelineState = "REMOVED"

	// StateUnknown is the new state of the pipeline.
	StateUnknown PipelineState = "UNKNOWN"
)

const tablePrefix = "hosts_v"
const validConditionType = "Valid"

// GetState returns the state of the pipeline.
func (instance *CyndiPipeline) GetState() PipelineState {
	switch {
	case instance.GetDeletionTimestamp() != nil:
		return StateRemoved
	case instance.Status.PipelineVersion == "":
		return StateNew
	case instance.IsValid():
		return StateValid
	case instance.Status.InitialSyncInProgress == true:
		return StateInitialSync
	case instance.GetValid() == metav1.ConditionFalse:
		return StateInvalid
	default:
		return StateUnknown
	}
}

// TransitionToNew transitions a pipeline to a new state.
func (instance *CyndiPipeline) TransitionToNew() error {
	instance.ResetValid()
	instance.Status.InitialSyncInProgress = false
	instance.Status.PipelineVersion = ""
	return nil
}

// TransitionToInitialSync transitions a pipeline to initial sync state.
func (instance *CyndiPipeline) TransitionToInitialSync(pipelineVersion string) error {
	if err := instance.assertState(StateInitialSync, StateNew); err != nil {
		return err
	}

	instance.ResetValid()
	instance.Status.InitialSyncInProgress = true
	instance.Status.PipelineVersion = pipelineVersion
	instance.Status.ConnectorName = ConnectorName(pipelineVersion, instance.Spec.AppName)
	instance.Status.TableName = TableName(pipelineVersion)

	return nil
}

// SetValid sets the status condition on the piprline.
func (instance *CyndiPipeline) SetValid(status metav1.ConditionStatus, reason string, message string, hostCount int64) {
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    validConditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})

	instance.Status.HostCount = hostCount

	switch status {
	case metav1.ConditionFalse:
		instance.Status.ValidationFailedCount++
	case metav1.ConditionUnknown:
		instance.Status.ValidationFailedCount = 0
	case metav1.ConditionTrue:
		instance.Status.ValidationFailedCount = 0
		instance.Status.InitialSyncInProgress = false
	}
}

// ResetValid resets the validation.
func (instance *CyndiPipeline) ResetValid() {
	instance.SetValid(metav1.ConditionUnknown, "New", "Validation not yet run", -1)
}

// IsValid checks the status condition is valid (present and equal)
func (instance *CyndiPipeline) IsValid() bool {
	return meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, validConditionType, metav1.ConditionTrue)
}

// GetValid returns the condition status, returning ConditionUnknown if the condition is nil.
func (instance *CyndiPipeline) GetValid() metav1.ConditionStatus {
	condition := meta.FindStatusCondition(instance.Status.Conditions, validConditionType)

	if condition == nil {
		return metav1.ConditionUnknown
	}

	return condition.Status
}

func (instance *CyndiPipeline) assertState(targetState PipelineState, validStates ...PipelineState) error {
	for _, state := range validStates {
		if instance.GetState() == state {
			return nil
		}
	}

	return fmt.Errorf("Attempted invalid state transition from %s to %s", instance.GetState(), targetState)
}

// TableName returns a formatted table name.
func TableName(pipelineVersion string) string {
	return fmt.Sprintf("%s%s", tablePrefix, pipelineVersion)
}

// TableNameToConnectorName returns the connector name for the app/table supplied.
func TableNameToConnectorName(tableName string, appName string) string {
	return ConnectorName(string(tableName[len(tablePrefix):]), appName)
}

// ConnectorName returns the cyndi table name given the pipeline version and appName.
func ConnectorName(pipelineVersion string, appName string) string {
	return fmt.Sprintf("cyndi-%s-%s", appName, strings.Replace(pipelineVersion, "_", "-", 1))
}
