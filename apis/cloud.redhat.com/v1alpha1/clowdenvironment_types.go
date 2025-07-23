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
	"context"
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/go-logr/logr"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// WebMode details the mode of operation of the Clowder Web Provider
// +kubebuilder:validation:Enum=none;operator;local
type WebMode string

// GatewayCertMode details the mode of operation of the Gateway Cert
// +kubebuilder:validation:Enum=self-signed;acme;none
type GatewayCertMode string

// WebImages defines optional container image overrides for the web provider components
type WebImages struct {
	// Mock entitlements image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used.
	Mocktitlements string `json:"mocktitlements,omitempty"`

	// Keycloak image -- default is 'quay.io/keycloak/keycloak:{KeycloakVersion}' unless overridden here
	Keycloak string `json:"keycloak,omitempty"`

	// Caddy image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used.
	Caddy string `json:"caddy,omitempty"`

	// Caddy Gateway image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used.
	CaddyGateway string `json:"caddyGateway,omitempty"`

	// Caddy Reverse Proxy image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used.
	CaddyProxy string `json:"caddyProxy,omitempty"`

	// Mock BOP image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used.
	MockBOP string `json:"mockBop,omitempty"`
}

// WebConfig configures the Clowder provider controlling the creation of web
// services and their probes.
type WebConfig struct {
	// The port that web services inside ClowdApp pods should be served on.
	Port int32 `json:"port"`

	// The private port that web services inside a ClowdApp should be served on.
	PrivatePort int32 `json:"privatePort,omitempty"`

	// The auth port that the web local mode will use with the AuthSidecar
	AuthPort int32 `json:"aiuthPort,omitempty"`

	// An api prefix path that pods will be instructed to use when setting up
	// their web server.
	APIPrefix string `json:"apiPrefix,omitempty"`

	// The mode of operation of the Web provider. The allowed modes are
	// (*_none_*/*_operator_*), and (*_local_*) which deploys keycloak and BOP.
	Mode WebMode `json:"mode"`

	// The URL of BOP - only used in (*_none_*/*_operator_*) mode.
	BOPURL string `json:"bopURL,omitempty"`

	// Ingress Class Name used only in (*_local_*) mode.
	IngressClass string `json:"ingressClass,omitempty"`

	// Optional keycloak version override -- used only in (*_local_*) mode -- if not set, a hard-coded default is used.
	KeycloakVersion string `json:"keycloakVersion,omitempty"`

	// Optionally use PVC storage for keycloak db
	KeycloakPVC bool `json:"keycloakPVC,omitempty"`

	// Optional images to use for web provider components -- only applies when running in (*_local_*) mode.
	Images WebImages `json:"images,omitempty"`

	// TLS sidecar enablement
	TLS TLS `json:"tls,omitempty"`

	// Gateway cert
	GatewayCert GatewayCert `json:"gatewayCert,omitempty"`
}

type GatewayCert struct {
	// Determines whether to enable the gateway cert, default is disabled
	Enabled bool `json:"enabled,omitempty"`

	// Determines the mode of certificate generation, either self-signed or acme
	CertMode GatewayCertMode `json:"certMode,omitempty"`

	// Determines a ConfigMap in the target namespace of the env which has ca.pem detailing the cert to use for mTLS verification
	LocalCAConfigMap string `json:"localCAConfigMap,omitempty"`

	// The email address used to register with Let's Encrypt for acme mode certs
	EmailAddress string `json:"emailAddress,omitempty"`
}

type TLS struct {
	Enabled     bool  `json:"enabled,omitempty"`
	Port        int32 `json:"port,omitempty"`
	PrivatePort int32 `json:"privatePort,omitempty"`
}

// MetricsMode details the mode of operation of the Clowder Metrics Provider
// +kubebuilder:validation:Enum=none;operator;app-interface
type MetricsMode string

type PrometheusConfig struct {
	// Determines whether to deploy prometheus in operator mode
	Deploy bool `json:"deploy,omitempty"`

	// Specify prometheus internal URL when in app-interface mode
	AppInterfaceInternalURL string `json:"appInterfaceInternalURL,omitempty"`
}

// MetricsConfig configures the Clowder provider controlling the creation of
// metrics services and their probes.
type MetricsConfig struct {
	// The port that metrics services inside ClowdApp pods should be served on.
	Port int32 `json:"port"`

	// A prefix path that pods will be instructed to use when setting up their
	// metrics server.
	Path string `json:"path,omitempty"`

	// The mode of operation of the Metrics provider. The allowed modes are
	//  (*_none_*), which disables metrics service generation, or
	// (*_operator_*) where services and probes are generated.
	// (*_app-interface_*) where services and probes are generated for app-interface.
	Mode MetricsMode `json:"mode"`

	// Prometheus specific configuration
	Prometheus PrometheusConfig `json:"prometheus,omitempty"`
}

// KafkaMode details the mode of operation of the Clowder Kafka Provider
// +kubebuilder:validation:Enum=ephem-msk;managed;operator;app-interface;local;none
type KafkaMode string

// KafkaClusterConfig defines options related to the Kafka cluster managed/monitored by Clowder
type KafkaClusterConfig struct {
	// Defines the kafka cluster name (default: <ClowdEnvironment Name>)
	Name string `json:"name,omitempty"`

	// The namespace the kafka cluster is expected to reside in (default: the environment's targetNamespace)
	Namespace string `json:"namespace,omitempty"`

	// Force TLS
	ForceTLS bool `json:"forceTLS,omitempty"`

	// The requested number of replicas for kafka/zookeeper. If unset, default is '1'
	// +kubebuilder:validation:Minimum:=1
	Replicas int32 `json:"replicas,omitempty"`

	// Persistent volume storage size. If unset, default is '1Gi'
	// Only applies when KafkaConfig.PVC is set to 'true'
	StorageSize string `json:"storageSize,omitempty"`

	// Delete persistent volume claim if the Kafka cluster is deleted
	// Only applies when KafkaConfig.PVC is set to 'true'
	DeleteClaim bool `json:"deleteClaim,omitempty"`

	// Version. If unset, default is '2.5.0'
	Version string `json:"version,omitempty"`

	// Config full options
	Config *map[string]string `json:"config,omitempty"`

	// JVM Options
	JVMOptions strimzi.KafkaSpecKafkaJvmOptions `json:"jvmOptions,omitempty"`

	// Resource Limits
	Resources strimzi.KafkaSpecKafkaResources `json:"resources,omitempty"`
}

// KafkaConnectClusterConfig defines options related to the Kafka Connect cluster managed/monitored by Clowder
type KafkaConnectClusterConfig struct {
	// Defines the kafka connect cluster name (default: <kafka cluster's name>)
	Name string `json:"name,omitempty"`

	// The namespace the kafka connect cluster is expected to reside in (default: the kafka cluster's namespace)
	Namespace string `json:"namespace,omitempty"`

	// The requested number of replicas for kafka connect. If unset, default is '1'
	// +kubebuilder:validation:Minimum:=1
	Replicas int32 `json:"replicas,omitempty"`

	// Version. If unset, default is '3.6.0'
	Version string `json:"version,omitempty"`

	// Image. If unset, default is 'quay.io/redhat-user-workloads/hcm-eng-prod-tenant/kafka-connect/kafka-connect:latest'
	Image string `json:"image,omitempty"`

	// Resource Limits
	Resources strimzi.KafkaConnectSpecResources `json:"resources,omitempty"`
}

// NamespacedName type to represent a real Namespaced Name
type NamespacedName struct {
	// Name defines the Name of a resource.
	Name string `json:"name"`

	// Namespace defines the Namespace of a resource.
	Namespace string `json:"namespace"`
}

// KafkaConfig configures the Clowder provider controlling the creation of
// Kafka instances.
type KafkaConfig struct {
	// The mode of operation of the Clowder Kafka Provider. Valid options are:
	// (*_operator_*) which provisions Strimzi resources and will configure
	// KafkaTopic CRs and place them in the Kafka cluster's namespace described in the configuration,
	// (*_app-interface_*) which simply passes the topic names through to the App's
	// cdappconfig.json and expects app-interface to have created the relevant
	// topics, and (*_local_*) where a small instance of Kafka is created in the desired cluster namespace
	// and configured to auto-create topics.
	Mode KafkaMode `json:"mode"`

	// EnableLegacyStrimzi disables TLS + user auth
	EnableLegacyStrimzi bool `json:"enableLegacyStrimzi,omitempty"`

	// If using the (*_local_*) or (*_operator_*) mode and PVC is set to true, this sets the provisioned
	// Kafka instance to use a PVC instead of emptyDir for its volumes.
	PVC bool `json:"pvc,omitempty"`

	// Defines options related to the Kafka cluster for this environment. Ignored for (*_local_*) mode.
	Cluster KafkaClusterConfig `json:"cluster,omitempty"`

	// Defines options related to the Kafka Connect cluster for this environment. Ignored for (*_local_*) mode.
	Connect KafkaConnectClusterConfig `json:"connect,omitempty"`

	// Defines the secret reference for the Managed Kafka mode. Only used in (*_managed_*) mode.
	ManagedSecretRef NamespacedName `json:"managedSecretRef,omitempty"`

	// Managed topic prefix for the managed cluster. Only used in (*_managed_*) mode.
	ManagedPrefix string `json:"managedPrefix,omitempty"`

	// Namespace that kafkaTopics should be written to for (*_msk_*) mode.
	TopicNamespace string `json:"topicNamespace,omitempty"`

	// Cluster annotation identifier for (*_msk_*) mode.
	ClusterAnnotation string `json:"clusterAnnotation,omitempty"`

	// (Deprecated) Defines the cluster name to be used by the Kafka Provider this will
	// be used in some modes to locate the Kafka instance.
	ClusterName string `json:"clusterName,omitempty"`

	// (Deprecated) The Namespace the cluster is expected to reside in. This is only used
	// in (*_app-interface_*) and (*_operator_*) modes.
	Namespace string `json:"namespace,omitempty"`

	// (Deprecated) The namespace that the Kafka Connect cluster is expected to reside in. This is only used
	// in (*_app-interface_*) and (*_operator_*) modes.
	ConnectNamespace string `json:"connectNamespace,omitempty"`

	// (Deprecated) Defines the kafka connect cluster name that is used in this environment.
	ConnectClusterName string `json:"connectClusterName,omitempty"`

	// (Deprecated) (Unused)
	Suffix string `json:"suffix,omitempty"`

	// Sets the replica count for ephem-msk mode for kafka connect
	KafkaConnectReplicaCount int `json:"kafkaConnectReplicaCount,omitempty"`
}

// DatabaseMode details the mode of operation of the Clowder Database Provider
// +kubebuilder:validation:Enum=shared;app-interface;local;none
type DatabaseMode string

// DatabaseConfig configures the Clowder provider controlling the creation of
// Database instances.
type DatabaseConfig struct {
	// The mode of operation of the Clowder Database Provider. Valid options are:
	// (*_app-interface_*) where the provider will pass through database credentials
	// found in the secret defined by the database name in the ClowdApp, and (*_local_*)
	// where the provider will spin up a local instance of the database.
	Mode DatabaseMode `json:"mode"`

	// Indicates where Clowder will fetch the database CA certificate bundle from. Currently only used in
	// (*_app-interface_*) mode. If none is specified, the AWS RDS combined CA bundle is used.
	// +kubebuilder:validation:Pattern=`^https?:\/\/.+$`
	CaBundleURL string `json:"caBundleURL,omitempty"`

	// If using the (*_local_*) mode and PVC is set to true, this instructs the local
	// Database instance to use a PVC instead of emptyDir for its volumes.
	PVC bool `json:"pvc,omitempty"`
}

// LoggingMode details the mode of operation of the Clowder Logging Provider
// +kubebuilder:validation:Enum=app-interface;null;none
type LoggingMode string

// LoggingConfig configures the Clowder provider controlling the creation of
// Logging instances.
type LoggingConfig struct {
	// The mode of operation of the Clowder Logging Provider. Valid options are:
	// (*_app-interface_*) where the provider will pass through cloudwatch credentials
	// to the app configuration, and (*_none_*) where no logging will be configured.
	Mode LoggingMode `json:"mode"`
}

// ServiceMeshMode just determines if we enable or disable the service mesh
// +kubebuilder:validation:Enum=enabled;disabled
type ServiceMeshMode string

// ServiceMeshConfig determines if this env should be part of a service mesh
// and, if enabled, configures the service mesh
type ServiceMeshConfig struct {
	Mode ServiceMeshMode `json:"mode,omitempty"`
}

type ObjectStoreImages struct {
	Minio string `json:"minio,omitempty"`
}

// ObjectStoreMode details the mode of operation of the Clowder ObjectStore
// Provider
// +kubebuilder:validation:Enum=minio;app-interface;none
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

	// Override the object store images
	Images ObjectStoreImages `json:"images,omitempty"`
}

type FeatureFlagsImages struct {
	Unleash     string `json:"unleash,omitempty"`
	UnleashEdge string `json:"unleashEdge,omitempty"`
}

// FeatureFlagsMode details the mode of operation of the Clowder FeatureFlags
// Provider
// +kubebuilder:validation:Enum=local;app-interface;none
// +kubebuilder:validation:Optional
type FeatureFlagsMode string

// FeatureFlagsConfig configures the Clowder provider controlling the creation of
// FeatureFlag instances.
type FeatureFlagsConfig struct {
	// The mode of operation of the Clowder FeatureFlag Provider. Valid options are:
	// (*_app-interface_*) where the provider will pass through credentials
	// to the app configuration, and (*_local_*) where a local Unleash instance will
	// be created.
	Mode FeatureFlagsMode `json:"mode,omitempty"`

	// If using the (*_local_*) mode and PVC is set to true, this instructs the local
	// Database instance to use a PVC instead of emptyDir for its volumes.
	PVC bool `json:"pvc,omitempty"`

	// Defines the secret containing the client access token, only used for (*_app-interface_*)
	// mode.
	CredentialRef NamespacedName `json:"credentialRef,omitempty"`

	// Defines the hostname for (*_app-interface_*) mode
	Hostname string `json:"hostname,omitempty"`

	// Defineds the port for (*_app-interface_*) mode
	Port int32 `json:"port,omitempty"`

	// Defines images used for the feature flags provider
	Images FeatureFlagsImages `json:"images,omitempty"`
}

// InMemoryMode details the mode of operation of the Clowder InMemoryDB
// Provider
// +kubebuilder:validation:Enum=redis;elasticache;none
type InMemoryMode string

// InMemoryDBConfig configures the Clowder provider controlling the creation of
// InMemoryDB instances.
type InMemoryDBConfig struct {
	// The mode of operation of the Clowder InMemory Provider. Valid options are:
	// (*_redis_*) where a local Minio instance will be created, and (*_elasticache_*)
	// which will search the namespace of the ClowdApp for a secret called 'elasticache'
	Mode InMemoryMode `json:"mode"`

	// This image is only used in the (*_redis_*) mode, as elsewhere it will try to
	// inspect for a secret for a hostname and credentials.
	Image string `json:"image,omitempty"`
}

// AutoScaler mode enabled or disabled the autoscaler. The key "keda" is deprecated but preserved for backwards compatibility
// +kubebuilder:validation:Enum={"none", "enabled", "keda"}
type AutoScalerMode string

// AutoScalerConfig configures the Clowder provider controlling the creation of
// AutoScaler configuration.
type AutoScalerConfig struct {
	// Enable the autoscaler feature
	Mode AutoScalerMode `json:"mode,omitempty"`
}

// Describes what amount of app config is mounted to the pod
// +kubebuilder:validation:Enum={"none", "app", "", "environment"}
type ConfigAccessMode string

type TestingConfig struct {
	// Defines the environment for iqe/smoke testing
	Iqe IqeConfig `json:"iqe,omitempty"`

	// The mode of operation of the testing Pod. Valid options are:
	// 'default', 'view' or 'edit'
	K8SAccessLevel K8sAccessLevel `json:"k8sAccessLevel"`

	// The mode of operation for access to outside app configs. Valid
	// options are:
	// (*_none_*) -- no app config is mounted to the pod
	// (*_app_*) -- only the ClowdApp's config is mounted to the pod
	// (*_environment_*) -- the config for all apps in the env are mounted
	ConfigAccess ConfigAccessMode `json:"configAccess"`
}

type IqeConfig struct {
	ImageBase string `json:"imageBase"`

	// A pass-through of a resource requirements in k8s ResourceRequirements
	// format. If omitted, the default resource requirements from the
	// ClowdEnvironment will be used.
	Resources core.ResourceRequirements `json:"resources,omitempty"`

	// Defines the secret reference for loading vault credentials into the IQE job
	VaultSecretRef NamespacedName `json:"vaultSecretRef,omitempty"`

	// Defines configurations related to UI testing containers
	UI IqeUIConfig `json:"ui,omitempty"`
}

type IqeUIConfig struct {
	// Defines configurations for selenium containers in this environment
	Selenium IqeUISeleniumConfig `json:"selenium,omitempty"`
}

type IqeUISeleniumConfig struct {
	// Defines the image used for selenium containers in this environment
	ImageBase string `json:"imageBase,omitempty"`

	// Defines the default image tag used for selenium containers in this environment
	DefaultImageTag string `json:"defaultImageTag,omitempty"`

	// Defines the resource requests/limits set on selenium containers
	Resources core.ResourceRequirements `json:"resources,omitempty"`
}

// ServiceConfig provides options for k8s Service resources
type ServiceConfig struct {
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;""
	Type string `json:"type"`
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
	ResourceDefaults core.ResourceRequirements `json:"resourceDefaults"`

	ServiceConfig ServiceConfig `json:"serviceConfig,omitempty"`

	// Disabled turns off reconciliation for this ClowdEnv
	Disabled bool `json:"disabled,omitempty"`
}

type TokenRefresherConfig struct {
	// Enables or disables token refresher sidecars
	Enabled bool `json:"enabled"`
	// Configurable image
	Image string `json:"image,omitempty"`
}

type OtelCollectorConfig struct {
	// Enable or disable otel collector sidecar
	Enabled bool `json:"enabled"`
	// Configurable image
	Image string `json:"image,omitempty"`
	// Configurable shared ConfigMap name (optional)
	ConfigMap string `json:"configMap,omitempty"`
	// Environment variables to be set in the otel collector container
	EnvVars []EnvVar `json:"envVars,omitempty"`
}

// EnvVar represents an environment variable present in a Container.
type EnvVar struct {
	// Name of the environment variable. Must be a C_IDENTIFIER.
	Name string `json:"name"`

	// Variable references $(VAR_NAME) are expanded using the previous defined
	// environment variables in the container and any service environment variables.
	// If a variable cannot be resolved, the reference in the input string will be
	// unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME).
	// Escaped references will never be expanded, regardless of whether the variable
	// exists or not.
	// +optional
	Value string `json:"value,omitempty"`

	// Source for the environment variable's value. Cannot be used if value is not empty.
	// +optional
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarSource represents a source for the value of an EnvVar.
type EnvVarSource struct {
	// Selects a key of a ConfigMap.
	// +optional
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`

	// Selects a key of a secret in the pod's namespace
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`

	// Selects a field of the pod: supports metadata.name, metadata.namespace,
	// metadata.labels['<KEY>'], metadata.annotations['<KEY>'], spec.nodeName,
	// spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.
	// +optional
	FieldRef *core.ObjectFieldSelector `json:"fieldRef,omitempty"`
}

// ConfigMapKeySelector selects a key from a ConfigMap.
type ConfigMapKeySelector struct {
	// The ConfigMap to select from.
	LocalObjectReference `json:",inline"`
	// The key to select.
	Key string `json:"key"`
	// Specify whether the ConfigMap or its key must be defined
	// +optional
	Optional *bool `json:"optional,omitempty"`
}

// SecretKeySelector selects a key from a Secret.
type SecretKeySelector struct {
	// The name of the secret in the pod's namespace to select from.
	LocalObjectReference `json:",inline"`
	// The key of the secret to select from.  Must be a valid secret key.
	Key string `json:"key"`
	// Specify whether the Secret or its key must be defined
	// +optional
	Optional *bool `json:"optional,omitempty"`
}

// LocalObjectReference contains enough information to let you locate the
// referenced object inside the same namespace.
type LocalObjectReference struct {
	// Name of the referent.
	// +optional
	Name string `json:"name,omitempty"`
}

type Sidecars struct {
	// Sets up Token Refresher configuration
	TokenRefresher TokenRefresherConfig `json:"tokenRefresher,omitempty"`
	// Sets up OpenTelemetry collector configuration
	OtelCollector OtelCollectorConfig `json:"otelCollector,omitempty"`
}

type DeploymentConfig struct {
	OmitPullPolicy bool `json:"omitPullPolicy,omitempty"`
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

	// Defines the Configuration for the Clowder FeatureFlags Provider.
	FeatureFlags FeatureFlagsConfig `json:"featureFlags,omitempty"`

	// Defines the Configuration for the Clowder ServiceMesh Provider.
	ServiceMesh ServiceMeshConfig `json:"serviceMesh,omitempty"`

	// Defines the pull secret to use for the service accounts.
	PullSecrets []NamespacedName `json:"pullSecrets,omitempty"`

	// Defines the environment for iqe/smoke testing
	Testing TestingConfig `json:"testing,omitempty"`

	// Defines the sidecar configuration
	Sidecars Sidecars `json:"sidecars,omitempty"`

	// Defines the autoscaler configuration
	AutoScaler AutoScalerConfig `json:"autoScaler,omitempty"`

	// Defines the Deployment provider options
	Deployment DeploymentConfig `json:"deployment,omitempty"`
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
	Conditions      []clusterv1.Condition `json:"conditions,omitempty"`
	TargetNamespace string                `json:"targetNamespace,omitempty"`
	Ready           bool                  `json:"ready,omitempty"`
	Deployments     EnvResourceStatus     `json:"deployments,omitempty"`
	Apps            []AppInfo             `json:"apps,omitempty"`
	Generation      int64                 `json:"generation,omitempty"`
	Hostname        string                `json:"hostname,omitempty"`
	Prometheus      PrometheusStatus      `json:"prometheus,omitempty"`
}

type EnvResourceStatus struct {
	ManagedDeployments int32 `json:"managedDeployments"`
	ReadyDeployments   int32 `json:"readyDeployments"`
	ManagedTopics      int32 `json:"managedTopics"`
	ReadyTopics        int32 `json:"readyTopics"`
}

// PrometheusStatus provides info on how to connect to Prometheus
type PrometheusStatus struct {
	ServerAddress string `json:"serverAddress"`
}

// AppInfo details information about a specific app.
type AppInfo struct {
	Name        string           `json:"name"`
	Deployments []DeploymentInfo `json:"deployments"`
}

// DeploymentInfo defailts information about a specific deployment.
type DeploymentInfo struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname,omitempty"`
	Port     int32  `json:"port,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=env
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.deployments.readyDeployments"
// +kubebuilder:printcolumn:name="Managed",type="integer",JSONPath=".status.deployments.managedDeployments"
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".status.targetNamespace"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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

func (i *ClowdEnvironment) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

func (i *ClowdEnvironment) SetConditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

// GetLabels returns a base set of labels relating to the ClowdEnvironment.
func (i *ClowdEnvironment) GetLabels() map[string]string {
	return map[string]string{
		"app": i.ObjectMeta.Name,
	}
}

// MakeOwnerReference defines the owner reference pointing to the ClowdApp resource.
func (i *ClowdEnvironment) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
		Controller: utils.TruePtr(),
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

// GetPrimaryLabel returns the primary label name use for igentification.
func (i *ClowdEnvironment) GetPrimaryLabel() string {
	return "env"
}

// GetClowdSAName returns the ServiceAccount Name for the App
func (i *ClowdEnvironment) GetClowdSAName() string {
	return fmt.Sprintf("%s-env", i.GetClowdName())
}

// GetUID returns ObjectMeta.UID
func (i *ClowdEnvironment) GetUID() types.UID {
	return i.ObjectMeta.UID
}

// GetDeploymentStatus returns the Status.Deployments member
func (i *ClowdEnvironment) GetDeploymentStatus() *EnvResourceStatus {
	return &i.Status.Deployments
}

// GenerateTargetNamespace gets a generated target namespace if one is not provided
func (i *ClowdEnvironment) GenerateTargetNamespace() string {
	return fmt.Sprintf("clowdenv-%s-%s", i.Name, utils.RandStringLower(6))
}

// IsReady returns true when all deployments are ready and the reconciliation is successful
func (i *ClowdEnvironment) IsReady() bool {
	conditionCheck := false

	for _, condition := range i.Status.Conditions {
		if condition.Type == ReconciliationSuccessful {
			conditionCheck = true
		}
	}

	return i.Status.Ready && conditionCheck
}

// ConvertDeprecatedKafkaSpec converts values from the old Kafka provider spec into the new format
func (i *ClowdEnvironment) ConvertDeprecatedKafkaSpec() {
	if i.Spec.Providers.Kafka.ClusterName != "" {
		i.Spec.Providers.Kafka.Cluster.Name = i.Spec.Providers.Kafka.ClusterName
	}

	if i.Spec.Providers.Kafka.Namespace != "" {
		i.Spec.Providers.Kafka.Cluster.Namespace = i.Spec.Providers.Kafka.Namespace
	}

	if i.Spec.Providers.Kafka.ConnectNamespace != "" {
		i.Spec.Providers.Kafka.Connect.Namespace = i.Spec.Providers.Kafka.ConnectNamespace
	}

	if i.Spec.Providers.Kafka.ConnectClusterName != "" {
		i.Spec.Providers.Kafka.Connect.Name = i.Spec.Providers.Kafka.ConnectClusterName
	}
}

// GetAppsInEnv populates the appList with a list of all apps in the ClowdEnvironment.
func (i *ClowdEnvironment) GetAppsInEnv(ctx context.Context, pClient client.Client) (*ClowdAppList, error) {

	appList := &ClowdAppList{}

	err := pClient.List(ctx, appList, client.MatchingFields{"spec.envName": i.Name})

	if err != nil {
		return appList, errors.Wrap("could not list apps", err)
	}

	return appList, nil
}

// GetAppsInEnv populates the appList with a list of all apps in the ClowdEnvironment.
func (i *ClowdEnvironment) GetNamespacesInEnv(ctx context.Context, pClient client.Client) ([]string, error) {

	var err error
	var appList *ClowdAppList

	if appList, err = i.GetAppsInEnv(ctx, pClient); err != nil {
		return nil, err
	}

	tmpNamespace := map[string]bool{}

	for _, app := range appList.Items {
		tmpNamespace[app.Namespace] = true
	}

	tmpNamespace[i.Status.TargetNamespace] = true

	namespaceList := []string{}

	for namespace := range tmpNamespace {
		namespaceList = append(namespaceList, namespace)
	}

	if i.Spec.Providers.Kafka.Cluster.Namespace != "" {
		namespaceList = append(namespaceList, i.Spec.Providers.Kafka.Cluster.Namespace)
	}

	return namespaceList, nil
}

// IsNodePort indicates whether or not services are configured as NodePort or not
func (i *ClowdEnvironment) IsNodePort() bool {
	return i.Spec.ServiceConfig.Type == "NodePort"
}

// GetClowdHostname gets the hostname for a particular environment
func (i *ClowdEnvironment) GenerateHostname(ctx context.Context, pClient client.Client, log logr.Logger, random bool) string {
	nn := types.NamespacedName{
		Name: "cluster",
	}

	var icGVK = schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Kind:    "Ingress",
		Version: "v1",
	}

	ic := &unstructured.Unstructured{}
	ic.SetGroupVersionKind(icGVK)

	err := pClient.Get(ctx, nn, ic)
	if err != nil {
		log.Info("Couldn't find cluster route resource, defaulting to env name" + err.Error())
		return i.Name
	}

	randomIdent := utils.RandStringLower(8)

	obj := ic.Object
	if obj["spec"] != nil {
		spec := obj["spec"].(map[string]interface{})
		domain := spec["domain"]
		if domain != "" {
			if random {
				return fmt.Sprintf("ee-%s.%s", randomIdent, domain)
			}
			return fmt.Sprintf("%s.%s", i.Name, domain)
		}
	}

	log.Info("Route resource didn't contain spec.domain, defaulting to env name")

	return i.Name
}
