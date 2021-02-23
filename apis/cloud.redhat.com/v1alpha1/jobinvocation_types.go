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

// JobInvocationSpec defines the desired state of JobInvocation
type JobInvocationSpec struct {
	// Name of the ClowdApp who owns the jobs
	AppName string `json:"appName"`

	// Jobs is the set of jobs to be run by the invocation
	Jobs []string `json:"jobs"`
}

type JobResult struct {
	Name      string `json:"name"`
	Completed bool   `json:"completed"`
	Attempts  int    `json:"attempts"`
	StdOut    string `json:"stdout"`
}

// JobInvocationStatus defines the observed state of JobInvocation
type JobInvocationStatus struct {
	Results []JobResult `json:"results"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JobInvocation is the Schema for the jobinvocations API
type JobInvocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobInvocationSpec   `json:"spec,omitempty"`
	Status JobInvocationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JobInvocationList contains a list of JobInvocation
type JobInvocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobInvocation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JobInvocation{}, &JobInvocationList{})
}
