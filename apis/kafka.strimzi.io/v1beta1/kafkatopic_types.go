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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KafkaTopicSpec defines the desired state of KafkaTopic
type KafkaTopicSpec struct {
	// A key/value pair describing the configuration of a particular topic.
	Config map[string]string `json:"config,omitempty"`

	// The requested number of partitions for this topic.
	Partitions *int32 `json:"partitions"`

	// The requested number of replicas for this topic.
	Replicas *int32 `json:"replicas,omitempty"`

	// The topic name.
	TopicName string `json:"topicName"`
}

// KafkaTopicStatus defines the observed state of KafkaTopic
type KafkaTopicStatus struct {
	// A list of k8s Conditions.
	Conditions []Condition `json:"conditions,omitempty"`

	// The observed generation of the Kafka resource.
	ObservedGeneration *int32 `json:"observedGeneration,omitempty"`
}

type Condition struct {
	// The last transition time of the resource.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`

	// The message of the last transition.
	Message string `json:"message,omitempty"`

	// The Reason for hte transition change.
	Reason string `json:"reason,omitempty"`

	// The status of the condition.
	Status string `json:"status,omitempty"`

	// The type of the condition.
	Type string `json:"type,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaTopic is the Schema for the kafkatopics API
type KafkaTopic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The KafkaTopicSpec specification defines a KafkaTopic.
	Spec   KafkaTopicSpec   `json:"spec,omitempty"`
	Status KafkaTopicStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaTopicList contains a list of KafkaTopic
type KafkaTopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// A list of KafkaTopic objects.
	Items []KafkaTopic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaTopic{}, &KafkaTopicList{})
}
