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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InsightsAppSpec defines the desired state of InsightsApp
type InsightsAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image     string                  `json:"image"`
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

// InsightsAppStatus defines the observed state of InsightsApp
type InsightsAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InsightsApp is the Schema for the insightsapps API
type InsightsApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InsightsAppSpec   `json:"spec,omitempty"`
	Status InsightsAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InsightsAppList contains a list of InsightsApp
type InsightsAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InsightsApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InsightsApp{}, &InsightsAppList{})
}
