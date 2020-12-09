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
	"fmt"
	"strings"

	"cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1/common"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// WebMode details the mode of operation of the Clowder Web Provider
// +kubebuilder:validation:Enum=none;operator
type WebMode string

// WebConfig configures the Clowder provider controlling the creation of web
// services and their probes.
type WebConfig struct {
	// The port that web services inside ClowdApp pods should be served on.
	// If omitted, defaults to 8000.
	Port int32 `json:"port,omitempty"`

	// An api prefix path that pods will be instructed to use when setting up
	// their web server.
	ApiPrefix string `json:"apiPrefix,omitempty"`

	// The mode of operation of the Web provider. The allowed modes are
	// (*_none_*), which disables web service generation, or (*_operator_*)
	// where services and probes are generated.
	Mode WebMode `json:"mode"`
}

// MetricsMode details the mode of operation of the Clowder Metrics Provider
// +kubebuilder:validation:Enum=none;operator
type MetricsMode string

// MetricsConfig configures the Clowder provider controlling the creation of
// metrics services and their probes.
type MetricsConfig struct {
	// The port that metrics services inside ClowdApp pods should be served on.
	// If omitted, defaults to 9000.
	Port int32 `json:"port,omitempty"`

	// A prefix path that pods will be instructed to use when setting up their
	// metrics server.
	Path string `json:"path,omitempty"`

	// The mode of operation of the Metrics provider. The allowed modes are
	//  (*_none_*), which disables metrics service generation, or
	// (*_operator_*) where services and probes are generated.
	Mode MetricsMode `json:"mode"`
}

// TODO: Other potential mode: saas

// KafkaMode details the mode of operation of the Clowder Kafka Provider
// +kubebuilder:validation:Enum=operator;app-interface;local
type KafkaMode string

// KafkaConfig configures the Clowder provider controlling the creation of
// Kafka instances.
type KafkaConfig struct {
	// Defines the cluster name to be used by the Kafka Provider this will
	// be used in some modes to locate the Kafka instance.
	ClusterName string `json:"clusterName"`

	// The Namespace the cluster is expected to reside in. This is only used
	// in (*_app-interface_*) and (*_operator_*) modes.
	Namespace string `json:"namespace"`

	// The mode of operation of the Clowder Kafka Provider. Valid options are:
	// (*_operator_*) which expects a Strimzi Kafka instance and will configure
	// KafkaTopic CRs and place them in the Namespace described in the configuration,
	// (*_app-interface_*) which simple passes the topic names through to the App's
	// cdappconfig.json and expects app-interface to have created the relevant
	// topics, and (*_local_*) where a small instance of Kafka is created in the Env
	// namespace and configured to auto-create topics.
	Mode KafkaMode `json:"mode"`

	// (Unused)
	Suffix string `json:"suffix,omitempty"`

	// If using the (*_local_*) mode and PVC is set to true, this instructs the local
	// Kafka instance to use a PVC instead of emptyDir for its volumes.
	PVC bool `json:"pvc,omitempty"`
}

// TODO: Other potential modes: RDS and Operator (e.g. CrunchyDB)

// DatabaseMode details the mode of operation of the Clowder Database Provider
// +kubebuilder:validation:Enum=app-interface;local
type DatabaseMode string

// DatabaseConfig configures the Clowder provider controlling the creation of
// Database instances.
type DatabaseConfig struct {
	// The mode of operation of the Clowder Database Provider. Valid options are:
	// (*_app-interface_*) where the provider will pass through database credentials
	// found in the secret defined by the database name in the ClowdApp, and (*_local_*)
	// where the provider will spin up a local instance of the database.
	Mode DatabaseMode `json:"mode"`

	// In (*_local_*) mode, the Image field is used to define the database image
	// for local database instances.
	Image string `json:"image"`

	// If using the (*_local_*) mode and PVC is set to true, this instructs the local
	// Database instance to use a PVC instead of emptyDir for its volumes.
	PVC bool `json:"pvc,omitempty"`
}

// TODO: Other potential modes: splunk, kafka

// LoggingMode details the mode of operation of the Clowder Logging Provider
type LoggingMode string

// LoggingConfig configures the Clowder provider controlling the creation of
// Logging instances.
type LoggingConfig struct {
	// The mode of operation of the Clowder Logging Provider. Valid options are:
	// (*_app-interface_*) where the provider will pass through cloudwatch credentials
	// to the app configuration, and (*_none_*) where no logging will be configured.
	Mode LoggingMode `json:"mode"`
}

// TODO: Other potential mode: ceph, S3

// ObjectStoreMode details the mode of operation of the Clowder ObjectStore
// Provider
// +kubebuilder:validation:Enum=minio;app-interface
type ObjectStoreMode string

// ObjectStoreConfig configures the Clowder provider controlling the creation of
// ObjectStore instances.
type ObjectStoreConfig struct {
	// The mode of operation of the Clowder ObjectStore Provider. Valid options are:
	// (*_app-interface_*) where the provider will pass through Amazon S3 credentials
	// to the app configuration, and (*_minio_*) where a local Minio instance will
	// be created.
	Mode ObjectStoreMode `json:"mode"`

	// Currently unused.
	Suffix string `json:"suffix,omitempty"`

	// If using the (*_local_*) mode and PVC is set to true, this instructs the local
	// Database instance to use a PVC instead of emptyDir for its volumes.
	PVC bool `json:"pvc,omitempty"`
}

// InMemoryMode details the mode of operation of the Clowder InMemoryDB
// Provider
// +kubebuilder:validation:Enum=redis;app-interface
type InMemoryMode string

// InMemoryDBConfig configures the Clowder provider controlling the creation of
// InMemoryDB instances.
type InMemoryDBConfig struct {
	// The mode of operation of the Clowder InMemory Provider. Valid options are:
	// (*_redis_*) where a local Minio instance will be created. This provider currently
	// has no mode for app-interface.
	Mode InMemoryMode `json:"mode"`

	// If using the (*_local_*) mode and PVC is set to true, this instructs the local
	// Database instance to use a PVC instead of emptyDir for its volumes.
	PVC bool `json:"pvc,omitempty"`
}

// ClowdEnvironmentSpec defines the desired state of ClowdEnvironment.
type ClowdEnvironmentSpec struct {
	// TargetNamespace describes the namespace where any generated environmental
	// resources should end up, this is particularly important in (*_local_*) mode.
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// A ProvidersConfig object, detailing the setup and configuration of all the
	// providers used in this ClowdEnvironment.
	Providers ProvidersConfig `json:"providers"`

	// Defines the default resource requirements in standard k8s format in the
	// event that they omitted from a PodSpec inside a ClowdApp.
	ResourceDefaults v1.ResourceRequirements `json:"resourceDefaults"`
}

// ProvidersConfig defines a group of providers configuration for a ClowdEnvironment.
type ProvidersConfig struct {
	// Defines the Configuration for the Clowder Database Provider.
	Database DatabaseConfig `json:"db,omitempty"`

	// Defines the Configuration for the Clowder InMemoryDB Provider.
	InMemoryDB InMemoryDBConfig `json:"inMemoryDb"`

	// Defines the Configuration for the Clowder Kafka Provider.
	Kafka KafkaConfig `json:"kafka"`

	// Defines the Configuration for the Clowder Logging Provider.
	Logging LoggingConfig `json:"logging"`

	// Defines the Configuration for the Clowder Metrics Provider.
	Metrics MetricsConfig `json:"metrics,omitempty"`

	// Defines the Configuration for the Clowder ObjectStore Provider.
	ObjectStore ObjectStoreConfig `json:"objectStore"`

	// Defines the Configuration for the Clowder Web Provider.
	Web WebConfig `json:"web,omitempty"`
}

// MinioStatus defines the status of a minio instance in local mode.
type MinioStatus struct {
	// A reference to standard k8s secret.
	Credentials core.SecretReference `json:"credentials"`

	// The hostname of a Minio instance.
	Hostname string `json:"hostname"`

	// The port number the Minio instance is to be served on.
	Port int32 `json:"port"`
}

// ClowdEnvironmentStatus defines the observed state of ClowdEnvironment
type ClowdEnvironmentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	TargetNamespace string                  `json:"targetNamespace"`
	Deployments     common.DeploymentStatus `json:"deployments"`
	Apps            []AppInfo               `json:"apps,omitempty"`
}

type AppInfo struct {
	Name        string           `json:"name"`
	Deployments []DeploymentInfo `json:"deployments"`
}

type DeploymentInfo struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname,omitempty"`
	Port     int32  `json:"port,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=env

// ClowdEnvironment is the Schema for the clowdenvironments API
type ClowdEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// A ClowdEnvironmentSpec object.
	Spec   ClowdEnvironmentSpec   `json:"spec,omitempty"`
	Status ClowdEnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ClowdEnvironmentList contains a list of ClowdEnvironment
type ClowdEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// A list of ClowdEnvironment objects.
	Items []ClowdEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClowdEnvironment{}, &ClowdEnvironmentList{})
}

// GetLabels returns a base set of labels relating to the ClowdEnvironment.
func (i *ClowdEnvironment) GetLabels() map[string]string {
	return map[string]string{"app": i.ObjectMeta.Name}
}

// MakeOwnerReference defines the owner reference pointing to the ClowdApp resource.
func (i *ClowdEnvironment) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
		Controller: utils.PointTrue(),
	}
}

// GetClowdNamespace returns the namespace of the ClowdApp object.
func (i *ClowdEnvironment) GetClowdNamespace() string {
	return i.Status.TargetNamespace
}

// GetClowdName returns the name of the ClowdApp object.
func (i *ClowdEnvironment) GetClowdName() string {
	return i.Name
}

// GetUID returns ObjectMeta.UID
func (i *ClowdEnvironment) GetUID() types.UID {
	return i.ObjectMeta.UID
}

// GetDeploymentStatus returns the Status.Deployments member
func (i *ClowdEnvironment) GetDeploymentStatus() *common.DeploymentStatus {
	return &i.Status.Deployments
}

// GenerateTargetNamespace gets a generated target namespace if one is not provided
func (i *ClowdEnvironment) GenerateTargetNamespace() string {
	return fmt.Sprintf("clowdenv-%s-%s", i.Name, strings.ToLower(utils.RandString(6)))
}
