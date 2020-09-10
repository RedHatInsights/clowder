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

type Address struct {
	Host string `json:"host,omitempty"`
	Port *int32 `json:"port,omitempty"`
}

type KafkaListener struct {
	Addresses        []Address `json:"addresses,omitempty"`
	BootstrapServers string    `json:"bootstrapServers,omitempty"`
	Type             string    `json:"type,omitempty"`
	Certificates     []string  `json:"certificates,omitempty"`
}

// KafkaTopicStatus defines the observed state of KafkaTopic
type KafkaStatus struct {
	Conditions         []Condition     `json:"conditions,omitempty"`
	ObservedGeneration *int32          `json:"observedGeneration,omitempty"`
	Listeners          []KafkaListener `json:"listeners,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Kafka is the Schema for the kafkatopics API
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status KafkaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaTopicList contains a list of KafkaTopic
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{})
}
