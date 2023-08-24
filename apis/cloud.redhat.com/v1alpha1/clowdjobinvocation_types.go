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

	"context"

	batchv1 "k8s.io/api/batch/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type JobConditionState string

const (
	JobInvoked  JobConditionState = "Invoked"
	JobComplete JobConditionState = "Complete"
	JobFailed   JobConditionState = "Failed"
)

type JobTestingSpec struct {
	// Iqe is the job spec to override defaults from the ClowdApp's
	// definition of the job
	Iqe IqeJobSpec `json:"iqe,omitempty"`
}

type IqeJobSpec struct {
	// By default, Clowder will set the image on the ClowdJob to be the
	// baseImage:name-of-iqe-plugin, but only the tag can be overridden here
	ImageTag string `json:"imageTag,omitempty"`

	// By default, Clowder will use the plugin name indicated in the ClowdApp's
	// spec.testing.iqePlugin field. A comma,separated,list of plugins can be supplied
	// here if you wish you override the plugins.
	IqePlugins string `json:"plugins,omitempty"`

	// Indiciates the presence of a selenium container
	// Note: currently not implemented
	UI IqeUISpec `json:"ui,omitempty"`

	// sets the pytest -m args
	Marker string `json:"marker,omitempty"`

	// sets value for ENV_FOR_DYNACONF
	DynaconfEnvName string `json:"dynaconfEnvName"`

	// sets pytest -k args
	Filter string `json:"filter,omitempty"`

	// Use to start the IQE pod without running tests and leave it up so that 'rsh' can be invoked
	Debug bool `json:"debug,omitempty"`

	// sets values passed to IQE '--requirements' arg
	Requirements *[]string `json:"requirements,omitempty"`

	// sets values passed to IQE '--requirements-priority' arg
	RequirementsPriority *[]string `json:"requirementsPriority,omitempty"`

	// sets values passed to IQE '--test-importance' arg
	TestImportance *[]string `json:"testImportance,omitempty"`

	// sets value for IQE_LOG_LEVEL (default if empty: "info")
	//+kubebuilder:validation:Enum={"", "critical", "error", "warning", "info", "debug", "notset"}
	LogLevel string `json:"logLevel,omitempty"`

	// sets value passed to IQE 'IQE_PARALLEL_ENABLED' arg
	ParallelEnabled string `json:"parallelEnabled,omitempty"`

	// sets value passed to IQE 'IQE_PARALLEL_WORKER_COUNT' arg
	ParallelWorkerCount string `json:"parallelWorkerCount,omitempty"`

	// sets value passed to IQE 'IQE_RP_ARGS' report portal args
	RpArgs string `json:"rpArgs,omitempty"`

	// sets value passed to IQE 'IQE_IBUTSU_SOURCE' args
	IbutsuSource string `json:"ibutsuSource,omitempty"`
}

type IqeUISpec struct {
	// No longer used
	Enabled bool `json:"enabled,omitempty"`

	// Configuration options for running IQE with a selenium container
	Selenium IqeSeleniumSpec `json:"selenium,omitempty"`
}

type IqeSeleniumSpec struct {
	// Whether or not a selenium container should be deployed in the IQE pod
	Deploy bool `json:"deploy,omitempty"`

	// Name of selenium image tag to use if not using the environment's default
	ImageTag string `json:"imageTag,omitempty"`
}

// ClowdJobInvocationSpec defines the desired state of ClowdJobInvocation
type ClowdJobInvocationSpec struct {
	// Name of the ClowdApp who owns the jobs
	AppName string `json:"appName"`

	// Jobs is the set of jobs to be run by the invocation
	Jobs []string `json:"jobs,omitempty"`

	// Testing is the struct for building out test jobs (iqe, etc) in a CJI
	Testing JobTestingSpec `json:"testing,omitempty"`

	// RunOnNotReady is a flag that when true, the job will not wait for the deployment to be ready to run
	RunOnNotReady bool `json:"runOnNotReady,omitempty"`
}

// ClowdJobInvocationStatus defines the observed state of ClowdJobInvocation
type ClowdJobInvocationStatus struct {
	// Completed is false and updated when all jobs have either finished
	// successfully or failed past their backoff and retry values
	Completed bool `json:"completed"`
	// DEPRECATED : Jobs is an array of jobs name run by a CJI.
	Jobs []string `json:"jobs,omitempty"`
	// JobMap is a map of the job names run by Job invocation and their outcomes
	JobMap     map[string]JobConditionState `json:"jobMap"`
	Conditions []clusterv1.Condition        `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cji
// +kubebuilder:printcolumn:name="Completed",type="boolean",JSONPath=".status.completed"

// ClowdJobInvocation is the Schema for the jobinvocations API
type ClowdJobInvocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClowdJobInvocationSpec   `json:"spec,omitempty"`
	Status ClowdJobInvocationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClowdJobInvocationList contains a list of ClowdJobInvocation
type ClowdJobInvocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClowdJobInvocation `json:"items"`
}

func (i *ClowdJobInvocation) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

func (i *ClowdJobInvocation) SetConditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&ClowdJobInvocation{}, &ClowdJobInvocationList{})
}

// GetLabels returns a base set of labels relating to the ClowdJobInvocation.
func (i *ClowdJobInvocation) GetLabels() map[string]string {
	if i.Labels == nil {
		i.Labels = map[string]string{}
	}

	if _, ok := i.Labels["clowdjob"]; !ok {
		i.Labels["clowdjob"] = i.ObjectMeta.Name
	}

	newMap := make(map[string]string, len(i.Labels))

	for k, v := range i.Labels {
		newMap[k] = v
	}

	return newMap
}

// GetNamespacedName contructs a new namespaced name for an object from the pattern.
func (i *ClowdJobInvocation) GetNamespacedName(pattern string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: i.Namespace,
		Name:      fmt.Sprintf(pattern, i.Name),
	}
}

// MakeOwnerReference defines the owner reference pointing to the ClowdJobInvocation resource.
func (i *ClowdJobInvocation) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
		Controller: utils.TruePtr(),
	}
}

// GetClowdNamespace returns the namespace of the ClowdJobInvocation object.
func (i *ClowdJobInvocation) GetClowdNamespace() string {
	return i.Namespace
}

// GetClowdName returns the name of the ClowdJobInvocation object.
func (i *ClowdJobInvocation) GetClowdName() string {
	return i.Name
}

// GetClowdName returns the name of the ClowdJobInvocation object.
func (i *ClowdJobInvocation) GetClowdSAName() string {
	return fmt.Sprintf("%s-cji", i.Name)
}

// GetIQEName returns the name of the ClowdJobInvocation's IQE job.
func (i *ClowdJobInvocation) GetIQEName() string {
	return fmt.Sprintf("%s-iqe", i.Name)
}

// GetUID returns ObjectMeta.UID
func (i *ClowdJobInvocation) GetUID() types.UID {
	return i.ObjectMeta.UID
}

// SetObjectMeta sets the metadata on a ClowdApp object.
func (i *ClowdJobInvocation) SetObjectMeta(o metav1.Object, opts ...omfunc) {
	o.SetName(i.Name)
	o.SetNamespace(i.Namespace)
	o.SetLabels(i.GetLabels())
	o.SetOwnerReferences([]metav1.OwnerReference{i.MakeOwnerReference()})

	for _, opt := range opts {
		opt(o)
	}
}

func (i *ClowdJobInvocation) GetInvokedJobs(ctx context.Context, c client.Client) (*batchv1.JobList, error) {

	jobs := batchv1.JobList{}
	if err := c.List(ctx, &jobs, client.InNamespace(i.ObjectMeta.Namespace)); err != nil {
		return nil, err
	}

	return &jobs, nil
}

func (i *ClowdJobInvocation) GenerateJobName() string {
	randomString := utils.RandStringLower(7)
	return fmt.Sprintf("%s-iqe-%s", i.Name, randomString)
}
