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
	"reflect"
	"time"

	"context"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	cond "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// used when desiring to run `oc debug`on the Job to cause pod to immediately & gracefully exit
	Debug bool `json:"debug,omitempty"`

	// sets values passed to IQE '--requirements' arg
	Requirements *[]string `json:"requirements,omitempty"`

	// sets values passed to IQE '--requirements-priority' arg
	RequirementsPriority *[]string `json:"requirementsPriority,omitempty"`

	// sets values passed to IQE '--test-importance' arg
	TestImportance *[]string `json:"testImportance,omitempty"`
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
		Controller: common.TruePtr(),
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

func (i *ClowdJobInvocation) GetJobsStatus(jobs *batchv1.JobList) bool {
	jobsRequired := len(i.Spec.Jobs)
	var emptyTesting IqeJobSpec
	if i.Spec.Testing.Iqe != emptyTesting {
		jobsRequired += 1
	}
	jobsCompleted := i.countCompletedJobs(jobs)
	return jobsCompleted == jobsRequired
}

func (i *ClowdJobInvocation) UpdateInvokedJobStatus(ctx context.Context, jobs *batchv1.JobList) error {

	for j := range i.Status.JobMap {
		for _, s := range jobs.Items {
			jobName := s.ObjectMeta.Name
			if j == jobName {
				if len(s.Status.Conditions) > 0 {
					condition := s.Status.Conditions[0].Type
					switch condition {
					case "Complete":
						i.Status.JobMap[jobName] = JobComplete
					case "Failed":
						i.Status.JobMap[jobName] = JobFailed
					default:
						i.Status.JobMap[jobName] = JobInvoked
					}
				}
			}
		}
	}
	return nil
}

func (i *ClowdJobInvocation) countCompletedJobs(jobs *batchv1.JobList) int {

	jobsCompleted := 0

	// A job either completes successfully, or fails to succeed within the
	// backoffLimit threshold. The Condition status is only populated when
	// the jobs have succeeded or passed the backoff limit
	for _, j := range jobs.Items {
		for s := range i.Status.JobMap {
			if s == j.ObjectMeta.Name {
				if len(j.Status.Conditions) > 0 {
					condition := j.Status.Conditions[0].Type
					if condition == "Complete" || condition == "Failed" {
						jobsCompleted++
					}
				}
			}

		}
	}
	return jobsCompleted
}

func (o *ClowdJobInvocation) SetClowdJobInvocationConditions(ctx context.Context, client client.Client, state clusterv1.ConditionType, err error) error {
	conditions := []clusterv1.Condition{}

	loopConditions := []clusterv1.ConditionType{ReconciliationSuccessful, ReconciliationFailed}
	for _, conditionType := range loopConditions {
		condition := &clusterv1.Condition{}
		condition.Type = conditionType
		condition.Status = core.ConditionFalse

		if state == conditionType {
			condition.Status = core.ConditionTrue
			if err != nil {
				condition.Reason = err.Error()
			}
		}

		condition.LastTransitionTime = v1.Now()
		conditions = append(conditions, *condition)
	}

	// Setup custom status for CJI
	condition := &clusterv1.Condition{}
	condition.Type = JobInvocationComplete
	condition.Status = core.ConditionFalse
	condition.Message = "Some Jobs are still incomplete"
	condition.LastTransitionTime = v1.Now()
	if err != nil {
		condition.Reason = err.Error()
	}

	jobs, err := o.GetInvokedJobs(ctx, client)
	if err != nil {
		return err
	}
	jobStatus := o.GetJobsStatus(jobs)

	if jobStatus {
		condition.Status = core.ConditionTrue
		condition.Message = "All ClowdJob invocations complete"
	}
	conditions = append(conditions, *condition)

	for _, condition := range conditions {
		cond.Set(o, &condition)
	}

	o.Status.Completed = jobStatus
	o.UpdateInvokedJobStatus(ctx, jobs)

	if err := client.Status().Update(ctx, o); err != nil {
		return err
	}
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1464#issuecomment-811930090
	// Handle the lag between the client and the k8s cache
	cjiState := o.Status
	nn := types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}
	if err := wait.Poll(100*time.Millisecond, 2*time.Second, func() (bool, error) {
		if err := client.Get(ctx, nn, o); err != nil {
			return false, fmt.Errorf("failed to get cji: %w", err)
		}
		return reflect.DeepEqual(o.Status, cjiState), nil
	}); err != nil {
		return fmt.Errorf("failed to wait for cached cji %s to get into state %s: %w", nn.String(), state, err)
	}
	return nil
}
