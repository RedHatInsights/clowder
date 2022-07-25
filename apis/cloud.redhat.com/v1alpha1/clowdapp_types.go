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
	"errors"
	"fmt"

	cerrors "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	keda "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InitContainer is a struct defining a k8s init container. This will be
// deployed along with the parent pod and is used to carry out one time
// initialization procedures.
type InitContainer struct {
	// Name gives an identifier in the situation where multiple init containers exist
	Name string `json:"name,omitempty"`

	// Image refers to the container image used to create the init container
	// (if different from the primary pod image).
	Image string `json:"image,omitempty"`

	// A list of commands to run inside the parent Pod.
	Command []string `json:"command,omitempty"`

	// A list of args to be passed to the init container.
	Args []string `json:"args,omitempty"`

	// If true, inheirts the environment variables from the parent pod.
	// specification
	InheritEnv bool `json:"inheritEnv,omitempty"`

	// A list of environment variables used only by the initContainer.
	Env []v1.EnvVar `json:"env,omitempty"`
}

// DatabaseSpec is a struct defining a database to be exposed to a ClowdApp.
type DatabaseSpec struct {
	// Defines the Version of the PostGreSQL database, defaults to 12.
	// +kubebuilder:validation:Enum:=10;12;13
	Version *int32 `json:"version,omitempty"`

	// Defines the Name of the database to be created. This will be used as the
	// name of the logical database inside the database server in (*_local_*) mode
	// and the name of the secret to be used for Database configuration in (*_app-interface_*) mode.
	Name string `json:"name,omitempty"`

	// Defines the Name of the app to share a database from
	SharedDBAppName string `json:"sharedDbAppName,omitempty"`

	// T-shirt size, one of small, medium, large
	// +kubebuilder:validation:Enum={"small", "medium", "large"}
	DBVolumeSize string `json:"dbVolumeSize,omitempty"`

	// T-shirt size, one of small, medium, large
	// +kubebuilder:validation:Enum={"small", "medium", "large"}
	DBResourceSize string `json:"dbResourceSize,omitempty"`
}

// Job defines a ClowdJob
// A Job struct will deploy as a CronJob if `schedule` is set
// and will deploy as a Job if it is not set. Unsupported fields
// will be dropped from Jobs
type Job struct {
	// Name defines identifier of the Job. This name will be used to name the
	// CronJob resource, the container will be name identically.
	Name string `json:"name"`

	// Defines the schedule for the job to run
	Schedule string `json:"schedule,omitempty"`

	// PodSpec defines a container running inside the CronJob.
	PodSpec PodSpec `json:"podSpec"`

	// Defines the restart policy for the CronJob, defaults to never
	RestartPolicy v1.RestartPolicy `json:"restartPolicy,omitempty"`

	// Defines the concurrency policy for the CronJob, defaults to Allow
	// Only applies to Cronjobs
	ConcurrencyPolicy batch.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// Only applies to Cronjobs
	Suspend *bool `json:"suspend,omitempty"`

	// The number of successful finished jobs to retain. Value must be non-negative integer.
	// Defaults to 3.
	// Only applies to Cronjobs
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// The number of failed finished jobs to retain. Value must be non-negative integer.
	// Defaults to 1.
	// Only applies to Cronjobs
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`

	// Defines the StartingDeadlineSeconds for the CronJob
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// The activeDeadlineSeconds for the Job or CronJob.
	// More info: https://kubernetes.io/docs/concepts/workloads/controllers/job/
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
}

// WebDeprecated defines a boolean flag to help distinguish from the newer WebServices
type WebDeprecated bool

// PublicWebService is the definition of the public web service. There can be only
// one public service managed by Clowder.
type PublicWebService struct {

	// Enabled describes if Clowder should enable the public service and provide the
	// configuration in the cdappconfig.
	Enabled bool `json:"enabled,omitempty"`

	// ApiPath describes the api path that will be configured to serve this backend from.
	ApiPath string `json:"apiPath,omitempty"`

	// WhitelistPaths define the paths that do not require authentication
	WhitelistPaths []string `json:"whitelistPaths,omitempty"`
}

// AppProtocol is used to define an appProtocol for Istio
// +kubebuilder:validation:Enum={"http", "http2", "https", "tcp", "tls", "grpc", "grpc-web", "mongo", "mysql", "redis"}
type AppProtocol string

// PrivateWebService is the definition of the private web service. There can be only
// one private service managed by Clowder.
type PrivateWebService struct {
	// Enabled describes if Clowder should enable the private service and provide the
	// configuration in the cdappconfig.
	Enabled bool `json:"enabled,omitempty"`

	// AppProtocol determines the protocol to be used for the private port, (defaults to http)
	AppProtocol AppProtocol `json:"appProtocol,omitempty"`
}

// MetricsWebService is the definition of the metrics web service. This is automatically
// enabled and the configuration here at the moment is included for completeness, as there
// are no configurable options.
type MetricsWebService struct {
}

// WebServices defines the structs for the three exposed web services: public,
// private and metrics.
type WebServices struct {
	Public  PublicWebService  `json:"public,omitempty"`
	Private PrivateWebService `json:"private,omitempty"`
	Metrics MetricsWebService `json:"metrics,omitempty"`
}

// K8sAccessLevel defines the access level for the deployment, one of 'default', 'view' or 'edit'
// +kubebuilder:validation:Enum={"default", "view", "", "edit"}
type K8sAccessLevel string

type DeploymentMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Deployment defines a service running inside a ClowdApp and will output a deployment resource.
// Only one container per pod is allowed and this is defined in the PodSpec attribute.
type Deployment struct {
	// Name defines the identifier of a Pod inside the ClowdApp. This name will
	// be used along side the name of the ClowdApp itself to form a <app>-<pod>
	// pattern which will be used for all other created resources and also for
	// some labels. It must be unique within a ClowdApp.
	Name string `json:"name"`

	// Defines the minimum replica count for the pod.
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// If set to true, creates a service on the webPort defined in
	// the ClowdEnvironment resource, along with the relevant liveness and
	// readiness probes.
	Web WebDeprecated `json:"web,omitempty"`

	WebServices WebServices `json:"webServices,omitempty"`

	// PodSpec defines a container running inside a ClowdApp.
	PodSpec PodSpec `json:"podSpec"`

	// K8sAccessLevel defines the level of access for this deployment
	K8sAccessLevel K8sAccessLevel `json:"k8sAccessLevel,omitempty"`

	// AutoScaler defines the configuration for the Keda auto scaler
	AutoScaler *AutoScaler `json:"autoScaler,omitempty"`

	AutoScalerSimple *AutoScalerSimple `json:"autoScalerSimple,omitempty"`

	// DeploymentStrategy allows the deployment strategy to be set only if the
	// deployment has no public service enabled
	DeploymentStrategy *DeploymentStrategy `json:"deploymentStrategy,omitempty"`

	Metadata DeploymentMetadata `json:"metadata,omitempty"`
}

type DeploymentStrategy struct {
	// PrivateStrategy allows a deployment that only uses a private port to set
	// the deployment strategy one of Recreate or Rolling, default for a
	// private service is Recreate. This is to enable a quicker roll out for
	// services that do not have public facing endpoints.
	PrivateStrategy apps.DeploymentStrategyType `json:"privateStrategy,omitempty"`
}

type Sidecar struct {
	// The name of the sidecar, only supported names allowed, (token-refresher)
	Name string `json:"name"`

	// Defines if the sidecar is enabled, defaults to False
	Enabled bool `json:"enabled"`
}

// PodSpec defines a container running inside a ClowdApp.
type PodSpec struct {

	// Image refers to the container image used to create the pod.
	Image string `json:"image,omitempty"`

	// A list of init containers used to perform at-startup operations.
	InitContainers []InitContainer `json:"initContainers,omitempty"`

	// The command that will be invoked inside the pod at startup.
	Command []string `json:"command,omitempty"`

	// A list of args to be passed to the pod container.
	Args []string `json:"args,omitempty"`

	// A list of environment variables in k8s defined format.
	Env []v1.EnvVar `json:"env,omitempty"`

	// A pass-through of a resource requirements in k8s ResourceRequirements
	// format. If omitted, the default resource requirements from the
	// ClowdEnvironment will be used.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// A pass-through of a Liveness Probe specification in standard k8s format.
	// If omitted, a standard probe will be setup point to the webPort defined
	// in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to
	// false.
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`

	// A pass-through of a Readiness Probe specification in standard k8s format.
	// If omitted, a standard probe will be setup point to the webPort defined
	// in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to
	// false.
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`

	// A pass-through of a list of Volumes in standa k8s format.
	Volumes []v1.Volume `json:"volumes,omitempty"`

	// A pass-through of a list of VolumesMounts in standa k8s format.
	VolumeMounts []v1.VolumeMount `json:"volumeMounts,omitempty"`

	// Lists the expected side cars, will be validated in the validating webhook
	Sidecars []Sidecar `json:"sidecars,omitempty"`

	// MachinePool allows the pod to be scheduled to a particular machine pool.
	MachinePool string `json:"machinePool,omitempty"`
}

// SimpleAutoScalerMetric defines a metric of either a value or utilization
type SimpleAutoScalerMetric struct {
	ScaleAtValue       string `json:"scaleAtValue,omitempty"`
	ScaleAtUtilization int32  `json:"scaleAtUtilization,omitempty"`
}

// SimpleAutoScalerReplicas defines the minimum and maximum replica counts for the auto scaler
type SimpleAutoScalerReplicas struct {
	Min int32 `json:"min"`
	Max int32 `json:"max"`
}

// SimpleAutoScaler defines a simple HPA with scaling for RAM and CPU by
// value and utilization thresholds, along with replica count limits
type AutoScalerSimple struct {
	Replicas SimpleAutoScalerReplicas `json:"replicas"`
	RAM      SimpleAutoScalerMetric   `json:"ram,omitempty"`
	CPU      SimpleAutoScalerMetric   `json:"cpu,omitempty"`
}

// AutoScaler defines the autoscaling parameters of a KEDA ScaledObject targeting the given deployment.
type AutoScaler struct {
	// PollingInterval is the interval (in seconds) to check each trigger on.
	// Default is 30 seconds.
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// CooldownPeriod is the interval (in seconds) to wait after the last trigger reported active before
	// scaling the deployment down. Default is 5 minutes (300 seconds).
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// MaxReplicaCount is the maximum number of replicas the scaler will scale the deployment to.
	// Default is 10.
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	Advanced *keda.AdvancedConfig `json:"advanced,omitempty"`
	// +optional
	Triggers []keda.ScaleTriggers `json:"triggers,omitempty"`
	// +optional
	Fallback *keda.Fallback `json:"fallback,omitempty"`
	// ExternalHPA allows replicas on deployments to be controlled by another resource, but will
	// not be allowed to fall under the minReplicas as set in the ClowdApp.
	ExternalHPA bool `json:"externalHPA,omitempty"`
}

// CyndiSpec is used to indicate whether a ClowdApp needs database syndication configured by the
// cyndi operator and exposes a limited set of cyndi configuration options
type CyndiSpec struct {
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	// +kubebuilder:validation:Pattern:="[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*"
	AppName string `json:"appName,omitempty"`

	InsightsOnly bool `json:"insightsOnly,omitempty"`
}

// KafkaTopicSpec defines the desired state of KafkaTopic
type KafkaTopicSpec struct {
	// we re-define this spec rather than use strimzi.KafkaTopicSpec so that a ClowdApp's topic
	// spec has:
	//   * partitions optional
	//   * replicas optional
	//   * topicName required

	// A key/value pair describing the configuration of a particular topic.
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// The requested number of partitions for this topic. If unset, default is '3'
	// +optional
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=200000
	Partitions int32 `json:"partitions,omitempty"`

	// The requested number of replicas for this topic. If unset, default is '3'
	// +optional
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=32767
	Replicas int32 `json:"replicas,omitempty"`

	// The requested name for this topic.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=249
	// +kubebuilder:validation:Pattern:="[a-zA-Z0-9\\._\\-]"
	TopicName string `json:"topicName"`
}

type TestingSpec struct {
	IqePlugin string `json:"iqePlugin"`
}

// ClowdAppSpec is the main specification for a single Clowder Application
// it defines n pods along with dependencies that are shared between them.
type ClowdAppSpec struct {
	// A list of deployments
	Deployments []Deployment `json:"deployments,omitempty"`

	// A list of jobs
	Jobs []Job `json:"jobs,omitempty"`

	// The name of the ClowdEnvironment resource that this ClowdApp will use as
	// its base. This does not mean that the ClowdApp needs to be placed in the
	// same directory as the targetNamespace of the ClowdEnvironment.
	EnvName string `json:"envName"`

	// A list of Kafka topics that will be created and made available to all
	// the pods listed in the ClowdApp.
	KafkaTopics []KafkaTopicSpec `json:"kafkaTopics,omitempty"`

	// The database specification defines a single database, the configuration
	// of which will be made available to all the pods in the ClowdApp.
	Database DatabaseSpec `json:"database,omitempty"`

	// A list of string names defining storage buckets. In certain modes,
	// defined by the ClowdEnvironment, Clowder will create those buckets.
	ObjectStore []string `json:"objectStore,omitempty"`

	// If inMemoryDb is set to true, Clowder will pass configuration
	// of an In Memory Database to the pods in the ClowdApp. This single
	// instance will be shared between all apps.
	InMemoryDB bool `json:"inMemoryDb,omitempty"`

	// If featureFlags is set to true, Clowder will pass configuration of a
	// FeatureFlags instance to the pods in the ClowdApp. This single
	// instance will be shared between all apps.
	FeatureFlags bool `json:"featureFlags,omitempty"`

	// A list of dependencies in the form of the name of the ClowdApps that are
	// required to be present for this ClowdApp to function.
	Dependencies []string `json:"dependencies,omitempty"`

	// A list of optional dependencies in the form of the name of the ClowdApps that are
	// will be added to the configuration when present.
	OptionalDependencies []string `json:"optionalDependencies,omitempty"`

	// Iqe plugin and other specifics
	Testing TestingSpec `json:"testing,omitempty"`

	// Configures 'cyndi' database syndication for this app. When the app's ClowdEnvironment has
	// the kafka provider set to (*_operator_*) mode, Clowder will configure a CyndiPipeline
	// for this app in the environment's kafka-connect namespace. When the kafka provider is in
	// (*_app-interface_*) mode, Clowder will check to ensure that a CyndiPipeline resource exists
	// for the application in the environment's kafka-connect namespace. For all other kafka
	// provider modes, this configuration option has no effect.
	Cyndi CyndiSpec `json:"cyndi,omitempty"`

	// Disabled turns off reconciliation for this ClowdApp
	Disabled bool `json:"disabled,omitempty"`
}

const (
	// Ready means all the deployments are ready
	DeploymentsReady clusterv1.ConditionType = "DeploymentsReady"
	// ReconciliationSuccessful represents status of successful reconciliation
	ReconciliationSuccessful clusterv1.ConditionType = "ReconciliationSuccessful"
	// ReconciliationFailed means the reconciliation failed
	ReconciliationFailed clusterv1.ConditionType = "ReconciliationFailed"
	// JobInvocationComplete means all the Jobs have finished
	JobInvocationComplete clusterv1.ConditionType = "JobInvocationComplete"
)

// ClowdAppStatus defines the observed state of ClowdApp
type ClowdAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// ClowdEnvironmentStatus defines the observed state of ClowdEnvironment
	Deployments AppResourceStatus     `json:"deployments,omitempty"`
	Ready       bool                  `json:"ready"`
	Conditions  []clusterv1.Condition `json:"conditions,omitempty"`
}

type AppResourceStatus struct {
	ManagedDeployments int32 `json:"managedDeployments"`
	ReadyDeployments   int32 `json:"readyDeployments"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=app
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.deployments.readyDeployments"
// +kubebuilder:printcolumn:name="Managed",type="integer",JSONPath=".status.deployments.managedDeployments"
// +kubebuilder:printcolumn:name="EnvName",type="string",JSONPath=".spec.envName"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ClowdApp is the Schema for the clowdapps API
type ClowdApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// A ClowdApp specification.
	Spec   ClowdAppSpec   `json:"spec,omitempty"`
	Status ClowdAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClowdAppList contains a list of ClowdApp
type ClowdAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// A list of ClowdApp Resources.
	Items []ClowdApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClowdApp{}, &ClowdAppList{})
}

func (i *ClowdApp) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

func (i *ClowdApp) SetConditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

// GetLabels returns a base set of labels relating to the ClowdApp.
func (i *ClowdApp) GetLabels() map[string]string {
	if i.Labels == nil {
		i.Labels = map[string]string{}
	}

	if _, ok := i.Labels["app"]; !ok {
		i.Labels["app"] = i.ObjectMeta.Name
	}

	newMap := make(map[string]string, len(i.Labels))

	for k, v := range i.Labels {
		newMap[k] = v
	}

	return newMap
}

// GetNamespacedName contructs a new namespaced name for an object from the pattern.
func (i *ClowdApp) GetNamespacedName(pattern string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: i.Namespace,
		Name:      fmt.Sprintf(pattern, i.Name),
	}
}

// GetIdent returns an ident <env>.<app> that should be unique across the cluster.
func (i *ClowdApp) GetIdent() string {
	return fmt.Sprintf("%v.%v", i.Spec.EnvName, i.Name)
}

// MakeOwnerReference defines the owner reference pointing to the ClowdApp resource.
func (i *ClowdApp) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
		Controller: utils.TruePtr(),
	}
}

// GetPrimaryLabel returns the primary label name use for identification.
func (i *ClowdApp) GetPrimaryLabel() string {
	return "app"
}

// GetClowdNamespace returns the namespace of the ClowdApp object.
func (i *ClowdApp) GetClowdNamespace() string {
	return i.Namespace
}

// GetClowdName returns the name of the ClowdApp object.
func (i *ClowdApp) GetClowdName() string {
	return i.Name
}

// GetUID returns ObjectMeta.UID
func (i *ClowdApp) GetUID() types.UID {
	return i.ObjectMeta.UID
}

// GetDeploymentStatus returns the Status.Deployments member
func (i *ClowdApp) GetDeploymentStatus() *AppResourceStatus {
	return &i.Status.Deployments
}

// GetDeploymentStatus returns the Status.Deployments member
func (i *ClowdApp) GetDeploymentNamespacedName(d *Deployment) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", i.Name, d.Name),
		Namespace: i.Namespace,
	}
}

// GetDeploymentStatus returns the Status.Deployments member
func (i *ClowdApp) GetCronJobNamespacedName(d *Job) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", i.Name, d.Name),
		Namespace: i.Namespace,
	}
}

// IsReady returns true when all deployments are ready and the reconciliation is successful
func (i *ClowdApp) IsReady() bool {
	conditionCheck := false

	for _, condition := range i.Status.Conditions {
		if condition.Type == ReconciliationSuccessful {
			conditionCheck = true
		}
	}

	return i.Status.Ready && conditionCheck
}

// GetClowdSAName returns the ServiceAccount Name for the App
func (i *ClowdApp) GetClowdSAName() string {
	return fmt.Sprintf("%s-app", i.GetClowdName())
}

// Omfunc is a utility function that performs an operation on a metav1.Object.
type omfunc func(o metav1.Object)

// SetObjectMeta sets the metadata on a ClowdApp object.
func (i *ClowdApp) SetObjectMeta(o metav1.Object, opts ...omfunc) {
	o.SetName(i.Name)
	o.SetNamespace(i.Namespace)
	o.SetLabels(i.GetLabels())
	o.SetOwnerReferences([]metav1.OwnerReference{i.MakeOwnerReference()})

	for _, opt := range opts {
		opt(o)
	}
}

// Name returns a function that sets the name of an object to that of the
// passed in string.
func Name(name string) omfunc {
	return func(o metav1.Object) {
		o.SetName(name)
	}
}

// Namespace returns a function that sets the namespace of an object to that of the
// passed in string.
func Namespace(namespace string) omfunc {
	return func(o metav1.Object) {
		o.SetNamespace(namespace)
	}
}

// Labels returns a function that sets the labels of an object to that of the passed in labels.
func Labels(labels map[string]string) omfunc {
	return func(o metav1.Object) {
		o.SetLabels(labels)
	}
}

// GetAppInSameEnv populates the appList with a list of all apps in the same ClowdEnvironment. The
// environment is inferred from the given app.
func GetAppInSameEnv(ctx context.Context, pClient client.Client, app *ClowdApp, appList *ClowdAppList) error {
	err := pClient.List(ctx, appList, client.MatchingFields{"spec.envName": app.Spec.EnvName})

	if err != nil {
		return err
		// return errors.New("Could not get app list")
	}

	return nil
}

// GetAppForDBInSameEnv returns a point to a ClowdApp that has the sharedDB referenced by the given
// ClowdApp.
func GetAppForDBInSameEnv(ctx context.Context, pClient client.Client, app *ClowdApp) (*ClowdApp, error) {
	appList := &ClowdAppList{}
	var refApp ClowdApp

	err := GetAppInSameEnv(ctx, pClient, app, appList)

	if err != nil {
		return nil, err
	}

	for _, iapp := range appList.Items {
		if iapp.Name == app.Spec.Database.SharedDBAppName {
			refApp = iapp
			return &refApp, nil
		}
	}
	return nil, errors.New("could not get app for db in env")
}

func (i *ClowdApp) GetOurEnv(ctx context.Context, pClient client.Client, env *ClowdEnvironment) error {
	return pClient.Get(ctx, types.NamespacedName{Name: i.Spec.EnvName}, env)
}

// GetAppsInEnv populates the appList with a list of all apps in the ClowdEnvironment.
func (i *ClowdApp) GetNamespacesInEnv(ctx context.Context, pClient client.Client) ([]string, error) {

	var env = &ClowdEnvironment{}
	var err error

	if err = i.GetOurEnv(ctx, pClient, env); err != nil {
		return nil, cerrors.Wrap("get our env: ", err)
	}

	var appList *ClowdAppList

	if appList, err = env.GetAppsInEnv(ctx, pClient); err != nil {
		return nil, cerrors.Wrap("get apps in env: ", err)
	}

	tmpNamespace := map[string]bool{}

	for _, app := range appList.Items {
		tmpNamespace[app.Namespace] = true
	}
	tmpNamespace[env.Status.TargetNamespace] = true

	namespaceList := []string{}

	for namespace := range tmpNamespace {
		namespaceList = append(namespaceList, namespace)
	}

	return namespaceList, nil
}
