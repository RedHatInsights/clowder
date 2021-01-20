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

	"cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1/common"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// InitContainer is a struct defining a k8s init container. This will be
// deployed along with the parent pod and is used to carry out one time
// initialization procedures.
type InitContainer struct {
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
	Version *int32 `json:"version,omitempty"`

	// Defines the Name of the datbase to be created. This will be used as the
	// name of the logical database inside the database server in (*_local_*) mode
	// and the name of the secret to be used for Database configuration in (*_app-interface_*) mode.
	Name string `json:"name,omitempty"`
}

// Job defines either a Job to be used in creating a Job via external means, or
// a CronJob, the difference is the presense of the schedule field.
type Job struct {
	// Name defines identifier of the Job. This name will be used to name the
	// CronJob resource, the container will be name identically.
	Name string `json:"name"`

	// Defines the schedule for the job to run
	Schedule string `json:"schedule"`

	// PodSpec defines a container running inside the CronJob.
	PodSpec PodSpec `json:"podSpec,omitempty"`

	// Defines the restart policy for the CronJob, defaults to never
	RestartPolicy v1.RestartPolicy `json:"restartPolicy,omitempty"`
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
	Web bool `json:"web,omitempty"`

	// PodSpec defines a container running inside a ClowdApp.
	PodSpec PodSpec `json:"podSpec,omitempty"`
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
	// If omited, a standard probe will be setup point to the webPort defined
	// in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to
	// false.
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`

	// A pass-through of a Readiness Probe specification in standard k8s format.
	// If omited, a standard probe will be setup point to the webPort defined
	// in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to
	// false.
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`

	// A pass-through of a list of Volumes in standa k8s format.
	Volumes []v1.Volume `json:"volumes,omitempty"`

	// A pass-through of a list of VolumesMounts in standa k8s format.
	VolumeMounts []v1.VolumeMount `json:"volumeMounts,omitempty"`
}

type PodSpecDeprecated struct {
	Name           string                  `json:"name"`
	Web            bool                    `json:"web,omitempty"`
	MinReplicas    *int32                  `json:"minReplicas,omitempty"`
	Image          string                  `json:"image,omitempty"`
	InitContainers []InitContainer         `json:"initContainers,omitempty"`
	Command        []string                `json:"command,omitempty"`
	Args           []string                `json:"args,omitempty"`
	Env            []v1.EnvVar             `json:"env,omitempty"`
	Resources      v1.ResourceRequirements `json:"resources,omitempty"`
	LivenessProbe  *v1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe *v1.Probe               `json:"readinessProbe,omitempty"`
	Volumes        []v1.Volume             `json:"volumes,omitempty"`
	VolumeMounts   []v1.VolumeMount        `json:"volumeMounts,omitempty"`
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

// ClowdAppSpec is the main specification for a single Clowder Application
// it defines n pods along with dependencies that are shared between them.
type ClowdAppSpec struct {
	// A list of deployments
	Deployments []Deployment `json:"deployments,omitempty"`

	// A list of jobs
	Jobs []Job `json:"jobs,omitempty"`

	// Deprecated
	Pods []PodSpecDeprecated `json:"pods,omitempty"`

	// The name of the ClowdEnvironment resource that this ClowdApp will use as
	// its base. This does not mean that the ClowdApp needs to be placed in the
	// same directory as the targetNamespace of the ClowdEnvironment.
	EnvName string `json:"envName"`

	// A list of Kafka topics that will be created and made available to all
	// the pods listed in the ClowdApp.
	KafkaTopics []strimzi.KafkaTopicSpec `json:"kafkaTopics,omitempty"`

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

	// Configures 'cyndi' database syndication for this app
	Cyndi CyndiSpec `json:"cyndi,omitempty"`
}

// ClowdAppStatus defines the observed state of ClowdApp
type ClowdAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// ClowdEnvironmentStatus defines the observed state of ClowdEnvironment
	Deployments common.DeploymentStatus `json:"deployments"`
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
	MultiName       string `json:"multiName,omitempty"`

	// A list of ClowdApp Resources.
	Items []ClowdApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClowdApp{}, &ClowdAppList{})
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
func (i *ClowdApp) MakeOwnerReference() []metav1.OwnerReference {
	return []metav1.OwnerReference{{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
		Controller: utils.PointTrue(),
	}}
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
func (i *ClowdApp) GetDeploymentStatus() *common.DeploymentStatus {
	return &i.Status.Deployments
}

func (i *ClowdApp) ConvertToNewShim() {
	deps := []Deployment{}
	for _, pod := range i.Spec.Pods {
		dep := Deployment{
			Name:        pod.Name,
			Web:         pod.Web,
			MinReplicas: pod.MinReplicas,
			PodSpec: PodSpec{
				Image:          pod.Image,
				InitContainers: pod.InitContainers,
				Command:        pod.Command,
				Args:           pod.Args,
				Env:            pod.Env,
				Resources:      pod.Resources,
				LivenessProbe:  pod.LivenessProbe,
				ReadinessProbe: pod.ReadinessProbe,
				Volumes:        pod.Volumes,
				VolumeMounts:   pod.VolumeMounts,
			},
		}
		deps = append(deps, dep)
	}
	i.Spec.Deployments = deps
}

// Omfunc is a utility function that performs an operation on a metav1.Object.
type omfunc func(o metav1.Object)

// SetObjectMeta sets the metadata on a ClowdApp object.
func (i *ClowdApp) SetObjectMeta(o metav1.Object, opts ...omfunc) {
	o.SetName(i.Name)
	o.SetNamespace(i.Namespace)
	o.SetLabels(i.GetLabels())
	o.SetOwnerReferences(i.MakeOwnerReference())

	for _, opt := range opts {
		opt(o)
	}
}

func (i *ClowdApp) GetCustomLabeler(labels map[string]string, nn types.NamespacedName, baseResource obj.ClowdObject) func(metav1.Object) {
	appliedLabels := baseResource.GetLabels()
	if labels != nil {
		for k, v := range labels {
			appliedLabels[k] = v
		}
	}
	return utils.MakeLabeler(nn, appliedLabels, baseResource)
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

// GetLabels returns a base set of labels relating to the ClowdApp.
func (i *ClowdAppList) GetLabels() map[string]string {
	return map[string]string{"app": i.GetClowdName()}
}

// GetNamespacedName contructs a new namespaced name for an object from the pattern.
func (i *ClowdAppList) GetNamespacedName(pattern string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: i.Items[0].Namespace,
		Name:      i.MultiName,
	}
}

// GetIdent returns an ident <env>.<app> that should be unique across the cluster.
func (i *ClowdAppList) GetIdent() string {
	return i.MultiName
}

// MakeOwnerReference defines the owner reference pointing to the ClowdApp resource.
func (i *ClowdAppList) MakeOwnerReference() []metav1.OwnerReference {
	oRefs := []metav1.OwnerReference{}
	for _, obj := range i.Items {
		oRefs = append(oRefs, metav1.OwnerReference{
			APIVersion: obj.APIVersion,
			Kind:       obj.Kind,
			Name:       obj.ObjectMeta.Name,
			UID:        obj.ObjectMeta.UID,
			Controller: utils.PointTrue(),
		})
	}
	return oRefs
}

// GetClowdNamespace returns the namespace of the ClowdApp object.
func (i *ClowdAppList) GetClowdNamespace() string {
	return i.Items[0].Namespace
}

// GetClowdName returns the name of the ClowdApp object.
func (i *ClowdAppList) GetClowdName() string {
	return i.Items[0].Spec.EnvName
}

// GetUID returns ObjectMeta.UID
func (i *ClowdAppList) GetUID() types.UID {
	return ""
}

// GetDeploymentStatus returns the Status.Deployments member
func (i *ClowdAppList) GetDeploymentStatus() *common.DeploymentStatus {
	return &common.DeploymentStatus{}
}

func (i *ClowdAppList) GetCustomLabeler(labels map[string]string, nn types.NamespacedName, baseResource obj.ClowdObject) func(metav1.Object) {
	appliedLabels := baseResource.GetLabels()
	if labels != nil {
		for k, v := range labels {
			appliedLabels[k] = v
		}
	}
	return utils.MakeLabeler(nn, appliedLabels, baseResource)
}
