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

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// JobConditionState describes the state a job is in
type JobConditionState string

const (
	// JobInvoked represents a job that has been invoked
	JobInvoked JobConditionState = "Invoked"
	// JobComplete represents a job that has completed successfully
	JobComplete JobConditionState = "Complete"
	// JobFailed represents a job that has failed
	JobFailed JobConditionState = "Failed"
)

// JobTestingSpec is the struct for building out test jobs (iqe, etc) in a CJI
type JobTestingSpec struct {
	// Iqe is the job spec to override defaults from the ClowdApp's
	// definition of the job
	Iqe IqeJobSpec `json:"iqe,omitempty"`
}

// IqeJobSpec defines the specification for IQE (Integration Quality Engineering) jobs
type IqeJobSpec struct {
	// Image tag to use for IQE container. By default, Clowder will set the image tag to be
	// baseImage:name-of-iqe-plugin, where baseImage is defined in the ClowdEnvironment. Only the tag can be overridden here.
	ImageTag string `json:"imageTag,omitempty"`

	// A comma,separated,list indicating IQE plugin(s) to run tests for. By default, Clowder will use the plugin name given on the ClowdApp's
	// spec.testing.iqePlugin field. Use this field if you wish you override the plugin list.
	IqePlugins string `json:"plugins,omitempty"`

	// Defines configuration for a selenium container (optional)
	UI IqeUISpec `json:"ui,omitempty"`

	// Specifies environment variables to set on the IQE container
	Env *[]core.EnvVar `json:"env,omitempty"`

	// Changes entrypoint to invoke 'iqe container-debug' so that container starts but does not run tests, allowing 'rsh' to be invoked
	Debug bool `json:"debug,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_MARKER_EXPRESSION env var on the IQE container
	Marker string `json:"marker,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets ENV_FOR_DYNACONF env var on the IQE container
	DynaconfEnvName string `json:"dynaconfEnvName,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_FILTER_EXPRESSION env var on the IQE container
	Filter string `json:"filter,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_REQUIREMENTS env var on the IQE container
	Requirements *[]string `json:"requirements,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_REQUIREMENTS_PRIORITY env var on the IQE container
	RequirementsPriority *[]string `json:"requirementsPriority,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_TEST_IMPORTANCE env var on the IQE container
	TestImportance *[]string `json:"testImportance,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_LOG_LEVEL env var on the IQE container
	//+kubebuilder:validation:Enum={"", "critical", "error", "warning", "info", "debug", "notset"}
	LogLevel string `json:"logLevel,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_PARALLEL_ENABLED env var on the IQE container
	ParallelEnabled string `json:"parallelEnabled,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_PARALLEL_WORKER_COUNT env var on the IQE container
	ParallelWorkerCount string `json:"parallelWorkerCount,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_RP_ARGS env var on the IQE container
	RpArgs string `json:"rpArgs,omitempty"`

	// (DEPRECATED, using 'env' now preferred) sets IQE_IBUTSU_SOURCE env var on the IQE container
	IbutsuSource string `json:"ibutsuSource,omitempty"`
}

// IqeUISpec defines configuration options for running IQE with UI components
type IqeUISpec struct {
	// No longer used
	Enabled bool `json:"enabled,omitempty"`

	// Configuration options for running IQE with a selenium container
	Selenium IqeSeleniumSpec `json:"selenium,omitempty"`
}

// IqeSeleniumSpec defines configuration options for running IQE with a selenium container
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

	// Disabled is a flag to turn off CJI(s) from running
	Disabled bool `json:"disabled,omitempty"`
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

// GetConditions returns the conditions for this ClowdJobInvocation
func (i *ClowdJobInvocation) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

// SetConditions updates the conditions for this ClowdJobInvocation
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
		i.Labels["clowdjob"] = i.Name
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
		Name:       i.Name,
		UID:        i.UID,
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

// GetClowdSAName returns the service account name for the ClowdJobInvocation object.
func (i *ClowdJobInvocation) GetClowdSAName() string {
	return fmt.Sprintf("%s-cji", i.Name)
}

// GetIQEName returns the name of the ClowdJobInvocation's IQE job.
func (i *ClowdJobInvocation) GetIQEName() string {
	return fmt.Sprintf("%s-iqe", i.Name)
}

// GetUID returns ObjectMeta.UID
func (i *ClowdJobInvocation) GetUID() types.UID {
	return i.UID
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

// GetInvokedJobs retrieves all jobs associated with this ClowdJobInvocation
func (i *ClowdJobInvocation) GetInvokedJobs(ctx context.Context, c client.Client) (*batchv1.JobList, error) {

	jobs := batchv1.JobList{}
	if err := c.List(ctx, &jobs, client.InNamespace(i.Namespace)); err != nil {
		return nil, err
	}

	return &jobs, nil
}

// GenerateJobName generates a random job name for the Job
func (i *ClowdJobInvocation) GenerateJobName() string {
	randomString := utils.RandStringLower(7)
	return fmt.Sprintf("%s-iqe-%s", i.Name, randomString)
}
