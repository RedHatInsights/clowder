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

// Address struct represents the physical connection details of a Kafka Server.
type Address struct {
	// Host defines the hostname of the Kafka server.
	Host string `json:"host,omitempty"`

	// Port defines the port of the Kafka server.
	Port *int32 `json:"port,omitempty"`
}

// KafkaListener represents a configured Kafka listener instance.
type KafkaListener struct {
	// A list of addresses that the Kafka instance is listening on.
	Addresses []Address `json:"addresses,omitempty"`

	// A bootstrap server that the Kafka client can initially conenct to.
	BootstrapServers string `json:"bootstrapServers,omitempty"`

	// The Kafka server type.
	Type string `json:"type,omitempty"`

	// A list of certificates to be used with this Kafka instance.
	Certificates []string `json:"certificates,omitempty"`
}

// KafkaTopicStatus defines the observed state of KafkaTopic
type KafkaStatus struct {
	// A list of k8s Conditions.
	Conditions []Condition `json:"conditions,omitempty"`

	// The observed generation of the Kafka resource.
	ObservedGeneration *int32 `json:"observedGeneration,omitempty"`

	// A list of KafkaListener objects.
	Listeners []KafkaListener `json:"listeners,omitempty"`
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

// KafkaList contains a list of instances.
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// A list of Kafka objects.
	Items []Kafka `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{})
}
