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
	core "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=none;operator
type WebMode string

type WebConfig struct {
	Port      int32   `json:"port,omitempty"`
	ApiPrefix string  `json:"apiPrefix,omitempty"`
	Mode      WebMode `json:"mode"`
}

// +kubebuilder:validation:Enum=none;operator
type MetricsMode string

type MetricsConfig struct {
	Port int32       `json:"port,omitempty"`
	Path string      `json:"path,omitempty"`
	Mode MetricsMode `json:"mode"`
}

// TODO: Other potential mode: saas

// +kubebuilder:validation:Enum=operator;app-interface;local
type KafkaMode string

type KafkaConfig struct {
	ClusterName string    `json:"clusterName"`
	Namespace   string    `json:"namespace"`
	Mode        KafkaMode `json:"mode"`
	Suffix      string    `json:"suffix,omitempty"`
	PVC         bool      `json:"pvc,omitempty"`
}

// TODO: Other potential modes: RDS and Operator (e.g. CrunchyDB)

// +kubebuilder:validation:Enum=app-interface;local
type DatabaseMode string

type DatabaseConfig struct {
	Mode  DatabaseMode `json:"mode"`
	Image string       `json:"image"`
	PVC   bool         `json:"pvc,omitempty"`
}

// TODO: Other potential modes: splunk, kafka

type LoggingMode string

type LoggingConfig struct {
	Mode LoggingMode `json:"mode"`
}

// TODO: Other potential mode: ceph, S3

// +kubebuilder:validation:Enum=minio;app-interface
type ObjectStoreMode string

type ObjectStoreConfig struct {
	Mode   ObjectStoreMode `json:"mode"`
	Suffix string          `json:"suffix,omitempty"`
	PVC    bool            `json:"pvc,omitempty"`
}

// +kubebuilder:validation:Enum=redis;app-interface
type InMemoryMode string

type InMemoryDBConfig struct {
	Mode InMemoryMode `json:"mode"`
	PVC  bool         `json:"pvc,omitempty"`
}

// ClowdEnvironmentSpec defines the desired state of ClowdEnvironment
type ClowdEnvironmentSpec struct {
	TargetNamespace  string                  `json:"targetNamespace"`
	Providers        ProvidersConfig         `json:"providers"`
	ResourceDefaults v1.ResourceRequirements `json:"resourceDefaults"`
}

type ProvidersConfig struct {
	Database    DatabaseConfig    `json:"db,omitempty"`
	InMemoryDB  InMemoryDBConfig  `json:"inMemoryDb"`
	Kafka       KafkaConfig       `json:"kafka"`
	Logging     LoggingConfig     `json:"logging"`
	Metrics     MetricsConfig     `json:"metrics,omitempty"`
	ObjectStore ObjectStoreConfig `json:"objectStore"`
	Web         WebConfig         `json:"web,omitempty"`
}

type MinioStatus struct {
	Credentials core.SecretReference `json:"credentials"`
	Hostname    string               `json:"hostname"`
	Port        int32                `json:"port"`
}

type ObjectStoreStatus struct {
	Minio   MinioStatus `json:"minio,omitempty"`
	Buckets []string    `json:"buckets"`
}

// ClowdEnvironmentStatus defines the observed state of ClowdEnvironment
type ClowdEnvironmentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ObjectStore ObjectStoreStatus `json:"objectStore"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=env

// ClowdEnvironment is the Schema for the clowdenvironments API
type ClowdEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClowdEnvironmentSpec   `json:"spec,omitempty"`
	Status ClowdEnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClowdEnvironmentList contains a list of ClowdEnvironment
type ClowdEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClowdEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClowdEnvironment{}, &ClowdEnvironmentList{})
}

func (i *ClowdEnvironment) GetLabels() map[string]string {
	return map[string]string{"app": i.ObjectMeta.Name}
}

func (i *ClowdEnvironment) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
	}
}

func (i *ClowdEnvironment) GetClowdNamespace() string {
	return i.Spec.TargetNamespace
}

func (i *ClowdEnvironment) GetClowdName() string {
	return i.Name
}
