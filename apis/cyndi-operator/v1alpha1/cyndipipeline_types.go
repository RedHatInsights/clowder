/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CyndiPipelineSpec defines the desired state of CyndiPipeline
type CyndiPipelineSpec struct {

	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	// +kubebuilder:validation:Required
	AppName string `json:"appName"`

	// +optional
	InsightsOnly bool `json:"insightsOnly"`

	// +optional
	// +kubebuilder:validation:MinLength:=1
	ConnectCluster *string `json:"connectCluster,omitempty"`

	// +optional
	// +kubebuilder:validation:Min:=0
	MaxAge *int64 `json:"maxAge,omitempty"`

	// +optional
	// +kubebuilder:validation:Min:=0
	// +kubebuilder:validation:Max:=100
	ValidationThreshold *int64 `json:"validationThreshold,omitempty"`

	// +optional
	// +kubebuilder:validation:MinLength:=1
	Topic *string `json:"topic,omitempty"`

	// +optional
	// +kubebuilder:validation:MinLength:=1
	DbSecret *string `json:"dbSecret,omitempty"`

	// +optional
	// +kubebuilder:validation:MinLength:=1
	InventoryDbSecret *string `json:"inventoryDbSecret,omitempty"`
}

// CyndiPipelineStatus defines the observed state of CyndiPipeline
type CyndiPipelineStatus struct {

	// +kubebuilder:validation:Minimum:=0
	ValidationFailedCount int64 `json:"validationFailedCount"`

	PipelineVersion string `json:"pipelineVersion"`
	ConnectorName   string `json:"cyndiPipelineName"`
	TableName       string `json:"tableName"`

	CyndiConfigVersion string `json:"cyndiConfigVersion"`

	InitialSyncInProgress bool `json:"initialSyncInProgress"`

	// Name of the database table that is currently backing the "inventory.hosts" view
	// May differ from TableName e.g. during a refresh
	ActiveTableName string `json:"activeTableName"`

	Conditions []metav1.Condition `json:"conditions"`

	HostCount int64 `json:"hostCount"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cyndi,categories=all
// +kubebuilder:printcolumn:name="App",type=string,JSONPath=`.spec.appName`
// +kubebuilder:printcolumn:name="Insights only",type=boolean,JSONPath=`.spec.insightsOnly`
// +kubebuilder:printcolumn:name="Active table",type=string,JSONPath=`.status.activeTableName`
// +kubebuilder:printcolumn:name="Valid",type=string,JSONPath=`.status.conditions[?(@.type == "Valid")].status`
// +kubebuilder:printcolumn:name="Host Count",type="integer",JSONPath=".status.hostCount"
// +kubebuilder:printcolumn:name="Initial sync",type=boolean,JSONPath=`.status.initialSyncInProgress`
// +kubebuilder:printcolumn:name="Validation failure count",type=integer,JSONPath=`.status.validationFailedCount`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CyndiPipeline is the Schema for the cyndipipelines API
type CyndiPipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CyndiPipelineSpec   `json:"spec,omitempty"`
	Status CyndiPipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CyndiPipelineList contains a list of CyndiPipeline
type CyndiPipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CyndiPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CyndiPipeline{}, &CyndiPipelineList{})
}
