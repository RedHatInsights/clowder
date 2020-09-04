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

type WebConfig struct {
	Port      int32  `json:"port"`
	ApiPrefix string `json:"apiPrefix"`
}

type MetricsConfig struct {
	Port int32  `json:"port"`
	Path string `json:"path"`
}

type KafkaConfig struct {
	ClusterName string `json:"clusterName"`
	Namespace   string `json:"namespace"`
	Provider    string `json:"provider"`
}

type DatabaseConfig struct {
	Provider string `json:"provider"`
	Image    string `json:"image"`
}

type LoggingConfig struct {
	Providers []string `json:"providers"`
}

// InsightsBaseSpec defines the desired state of InsightsBase
type InsightsBaseSpec struct {
	Web      WebConfig      `json:"web,omitempty"`
	Metrics  MetricsConfig  `json:"metrics,omitempty"`
	Kafka    KafkaConfig    `json:"kafka"`
	Database DatabaseConfig `json:"db,omitempty"`
	Logging  LoggingConfig  `json:"logging"`
}

// InsightsBaseStatus defines the observed state of InsightsBase
type InsightsBaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InsightsBase is the Schema for the insightsbases API
type InsightsBase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InsightsBaseSpec   `json:"spec,omitempty"`
	Status InsightsBaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InsightsBaseList contains a list of InsightsBase
type InsightsBaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InsightsBase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InsightsBase{}, &InsightsBaseList{})
}
