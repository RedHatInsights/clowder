// +build !ignore_autogenerated

/*
Copyright 2021.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppInfo) DeepCopyInto(out *AppInfo) {
	*out = *in
	if in.Deployments != nil {
		in, out := &in.Deployments, &out.Deployments
		*out = make([]DeploymentInfo, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppInfo.
func (in *AppInfo) DeepCopy() *AppInfo {
	if in == nil {
		return nil
	}
	out := new(AppInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdApp) DeepCopyInto(out *ClowdApp) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdApp.
func (in *ClowdApp) DeepCopy() *ClowdApp {
	if in == nil {
		return nil
	}
	out := new(ClowdApp)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClowdApp) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdAppList) DeepCopyInto(out *ClowdAppList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClowdApp, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdAppList.
func (in *ClowdAppList) DeepCopy() *ClowdAppList {
	if in == nil {
		return nil
	}
	out := new(ClowdAppList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClowdAppList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdAppSpec) DeepCopyInto(out *ClowdAppSpec) {
	*out = *in
	if in.Deployments != nil {
		in, out := &in.Deployments, &out.Deployments
		*out = make([]Deployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Jobs != nil {
		in, out := &in.Jobs, &out.Jobs
		*out = make([]Job, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.KafkaTopics != nil {
		in, out := &in.KafkaTopics, &out.KafkaTopics
		*out = make([]KafkaTopicSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Database.DeepCopyInto(&out.Database)
	if in.ObjectStore != nil {
		in, out := &in.ObjectStore, &out.ObjectStore
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.OptionalDependencies != nil {
		in, out := &in.OptionalDependencies, &out.OptionalDependencies
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Testing = in.Testing
	out.Cyndi = in.Cyndi
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdAppSpec.
func (in *ClowdAppSpec) DeepCopy() *ClowdAppSpec {
	if in == nil {
		return nil
	}
	out := new(ClowdAppSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdAppStatus) DeepCopyInto(out *ClowdAppStatus) {
	*out = *in
	out.Deployments = in.Deployments
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ClowdCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdAppStatus.
func (in *ClowdAppStatus) DeepCopy() *ClowdAppStatus {
	if in == nil {
		return nil
	}
	out := new(ClowdAppStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdCondition) DeepCopyInto(out *ClowdCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdCondition.
func (in *ClowdCondition) DeepCopy() *ClowdCondition {
	if in == nil {
		return nil
	}
	out := new(ClowdCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdEnvironment) DeepCopyInto(out *ClowdEnvironment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdEnvironment.
func (in *ClowdEnvironment) DeepCopy() *ClowdEnvironment {
	if in == nil {
		return nil
	}
	out := new(ClowdEnvironment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClowdEnvironment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdEnvironmentList) DeepCopyInto(out *ClowdEnvironmentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClowdEnvironment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdEnvironmentList.
func (in *ClowdEnvironmentList) DeepCopy() *ClowdEnvironmentList {
	if in == nil {
		return nil
	}
	out := new(ClowdEnvironmentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClowdEnvironmentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdEnvironmentSpec) DeepCopyInto(out *ClowdEnvironmentSpec) {
	*out = *in
	in.Providers.DeepCopyInto(&out.Providers)
	in.ResourceDefaults.DeepCopyInto(&out.ResourceDefaults)
	out.ServiceConfig = in.ServiceConfig
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdEnvironmentSpec.
func (in *ClowdEnvironmentSpec) DeepCopy() *ClowdEnvironmentSpec {
	if in == nil {
		return nil
	}
	out := new(ClowdEnvironmentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdEnvironmentStatus) DeepCopyInto(out *ClowdEnvironmentStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ClowdCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Deployments = in.Deployments
	if in.Apps != nil {
		in, out := &in.Apps, &out.Apps
		*out = make([]AppInfo, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdEnvironmentStatus.
func (in *ClowdEnvironmentStatus) DeepCopy() *ClowdEnvironmentStatus {
	if in == nil {
		return nil
	}
	out := new(ClowdEnvironmentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdJobInvocation) DeepCopyInto(out *ClowdJobInvocation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdJobInvocation.
func (in *ClowdJobInvocation) DeepCopy() *ClowdJobInvocation {
	if in == nil {
		return nil
	}
	out := new(ClowdJobInvocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClowdJobInvocation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdJobInvocationList) DeepCopyInto(out *ClowdJobInvocationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClowdJobInvocation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdJobInvocationList.
func (in *ClowdJobInvocationList) DeepCopy() *ClowdJobInvocationList {
	if in == nil {
		return nil
	}
	out := new(ClowdJobInvocationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClowdJobInvocationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdJobInvocationSpec) DeepCopyInto(out *ClowdJobInvocationSpec) {
	*out = *in
	if in.Jobs != nil {
		in, out := &in.Jobs, &out.Jobs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Testing = in.Testing
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdJobInvocationSpec.
func (in *ClowdJobInvocationSpec) DeepCopy() *ClowdJobInvocationSpec {
	if in == nil {
		return nil
	}
	out := new(ClowdJobInvocationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClowdJobInvocationStatus) DeepCopyInto(out *ClowdJobInvocationStatus) {
	*out = *in
	if in.Jobs != nil {
		in, out := &in.Jobs, &out.Jobs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.JobMap != nil {
		in, out := &in.JobMap, &out.JobMap
		*out = make(map[string]JobConditionState, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ClowdCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClowdJobInvocationStatus.
func (in *ClowdJobInvocationStatus) DeepCopy() *ClowdJobInvocationStatus {
	if in == nil {
		return nil
	}
	out := new(ClowdJobInvocationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CyndiSpec) DeepCopyInto(out *CyndiSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CyndiSpec.
func (in *CyndiSpec) DeepCopy() *CyndiSpec {
	if in == nil {
		return nil
	}
	out := new(CyndiSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseConfig) DeepCopyInto(out *DatabaseConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseConfig.
func (in *DatabaseConfig) DeepCopy() *DatabaseConfig {
	if in == nil {
		return nil
	}
	out := new(DatabaseConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseSpec) DeepCopyInto(out *DatabaseSpec) {
	*out = *in
	if in.Version != nil {
		in, out := &in.Version, &out.Version
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseSpec.
func (in *DatabaseSpec) DeepCopy() *DatabaseSpec {
	if in == nil {
		return nil
	}
	out := new(DatabaseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Deployment) DeepCopyInto(out *Deployment) {
	*out = *in
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	out.WebServices = in.WebServices
	in.PodSpec.DeepCopyInto(&out.PodSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Deployment.
func (in *Deployment) DeepCopy() *Deployment {
	if in == nil {
		return nil
	}
	out := new(Deployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentInfo) DeepCopyInto(out *DeploymentInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentInfo.
func (in *DeploymentInfo) DeepCopy() *DeploymentInfo {
	if in == nil {
		return nil
	}
	out := new(DeploymentInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FeatureFlagsConfig) DeepCopyInto(out *FeatureFlagsConfig) {
	*out = *in
	out.CredentialRef = in.CredentialRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FeatureFlagsConfig.
func (in *FeatureFlagsConfig) DeepCopy() *FeatureFlagsConfig {
	if in == nil {
		return nil
	}
	out := new(FeatureFlagsConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryDBConfig) DeepCopyInto(out *InMemoryDBConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryDBConfig.
func (in *InMemoryDBConfig) DeepCopy() *InMemoryDBConfig {
	if in == nil {
		return nil
	}
	out := new(InMemoryDBConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InitContainer) DeepCopyInto(out *InitContainer) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InitContainer.
func (in *InitContainer) DeepCopy() *InitContainer {
	if in == nil {
		return nil
	}
	out := new(InitContainer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IqeConfig) DeepCopyInto(out *IqeConfig) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
	out.VaultSecretRef = in.VaultSecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IqeConfig.
func (in *IqeConfig) DeepCopy() *IqeConfig {
	if in == nil {
		return nil
	}
	out := new(IqeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IqeJobSpec) DeepCopyInto(out *IqeJobSpec) {
	*out = *in
	out.UI = in.UI
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IqeJobSpec.
func (in *IqeJobSpec) DeepCopy() *IqeJobSpec {
	if in == nil {
		return nil
	}
	out := new(IqeJobSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Job) DeepCopyInto(out *Job) {
	*out = *in
	in.PodSpec.DeepCopyInto(&out.PodSpec)
	if in.Suspend != nil {
		in, out := &in.Suspend, &out.Suspend
		*out = new(bool)
		**out = **in
	}
	if in.SuccessfulJobsHistoryLimit != nil {
		in, out := &in.SuccessfulJobsHistoryLimit, &out.SuccessfulJobsHistoryLimit
		*out = new(int32)
		**out = **in
	}
	if in.FailedJobsHistoryLimit != nil {
		in, out := &in.FailedJobsHistoryLimit, &out.FailedJobsHistoryLimit
		*out = new(int32)
		**out = **in
	}
	if in.StartingDeadlineSeconds != nil {
		in, out := &in.StartingDeadlineSeconds, &out.StartingDeadlineSeconds
		*out = new(int64)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Job.
func (in *Job) DeepCopy() *Job {
	if in == nil {
		return nil
	}
	out := new(Job)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobTestingSpec) DeepCopyInto(out *JobTestingSpec) {
	*out = *in
	out.Iqe = in.Iqe
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobTestingSpec.
func (in *JobTestingSpec) DeepCopy() *JobTestingSpec {
	if in == nil {
		return nil
	}
	out := new(JobTestingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaClusterConfig) DeepCopyInto(out *KafkaClusterConfig) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(map[string]string)
		if **in != nil {
			in, out := *in, *out
			*out = make(map[string]string, len(*in))
			for key, val := range *in {
				(*out)[key] = val
			}
		}
	}
	in.JVMOptions.DeepCopyInto(&out.JVMOptions)
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaClusterConfig.
func (in *KafkaClusterConfig) DeepCopy() *KafkaClusterConfig {
	if in == nil {
		return nil
	}
	out := new(KafkaClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaConfig) DeepCopyInto(out *KafkaConfig) {
	*out = *in
	in.Cluster.DeepCopyInto(&out.Cluster)
	out.Connect = in.Connect
	out.ManagedSecretRef = in.ManagedSecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaConfig.
func (in *KafkaConfig) DeepCopy() *KafkaConfig {
	if in == nil {
		return nil
	}
	out := new(KafkaConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaConnectClusterConfig) DeepCopyInto(out *KafkaConnectClusterConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaConnectClusterConfig.
func (in *KafkaConnectClusterConfig) DeepCopy() *KafkaConnectClusterConfig {
	if in == nil {
		return nil
	}
	out := new(KafkaConnectClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaTopicSpec) DeepCopyInto(out *KafkaTopicSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(v1beta1.KafkaTopicSpecConfig, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaTopicSpec.
func (in *KafkaTopicSpec) DeepCopy() *KafkaTopicSpec {
	if in == nil {
		return nil
	}
	out := new(KafkaTopicSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoggingConfig) DeepCopyInto(out *LoggingConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoggingConfig.
func (in *LoggingConfig) DeepCopy() *LoggingConfig {
	if in == nil {
		return nil
	}
	out := new(LoggingConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsConfig) DeepCopyInto(out *MetricsConfig) {
	*out = *in
	out.Prometheus = in.Prometheus
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsConfig.
func (in *MetricsConfig) DeepCopy() *MetricsConfig {
	if in == nil {
		return nil
	}
	out := new(MetricsConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsWebService) DeepCopyInto(out *MetricsWebService) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsWebService.
func (in *MetricsWebService) DeepCopy() *MetricsWebService {
	if in == nil {
		return nil
	}
	out := new(MetricsWebService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MinioStatus) DeepCopyInto(out *MinioStatus) {
	*out = *in
	out.Credentials = in.Credentials
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MinioStatus.
func (in *MinioStatus) DeepCopy() *MinioStatus {
	if in == nil {
		return nil
	}
	out := new(MinioStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespacedName) DeepCopyInto(out *NamespacedName) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespacedName.
func (in *NamespacedName) DeepCopy() *NamespacedName {
	if in == nil {
		return nil
	}
	out := new(NamespacedName)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectStoreConfig) DeepCopyInto(out *ObjectStoreConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectStoreConfig.
func (in *ObjectStoreConfig) DeepCopy() *ObjectStoreConfig {
	if in == nil {
		return nil
	}
	out := new(ObjectStoreConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodSpec) DeepCopyInto(out *PodSpec) {
	*out = *in
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]InitContainer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = make([]Sidecar, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodSpec.
func (in *PodSpec) DeepCopy() *PodSpec {
	if in == nil {
		return nil
	}
	out := new(PodSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrivateWebService) DeepCopyInto(out *PrivateWebService) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrivateWebService.
func (in *PrivateWebService) DeepCopy() *PrivateWebService {
	if in == nil {
		return nil
	}
	out := new(PrivateWebService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrometheusConfig) DeepCopyInto(out *PrometheusConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrometheusConfig.
func (in *PrometheusConfig) DeepCopy() *PrometheusConfig {
	if in == nil {
		return nil
	}
	out := new(PrometheusConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProvidersConfig) DeepCopyInto(out *ProvidersConfig) {
	*out = *in
	out.Database = in.Database
	out.InMemoryDB = in.InMemoryDB
	in.Kafka.DeepCopyInto(&out.Kafka)
	out.Logging = in.Logging
	out.Metrics = in.Metrics
	out.ObjectStore = in.ObjectStore
	out.Web = in.Web
	out.FeatureFlags = in.FeatureFlags
	out.ServiceMesh = in.ServiceMesh
	if in.PullSecrets != nil {
		in, out := &in.PullSecrets, &out.PullSecrets
		*out = make([]NamespacedName, len(*in))
		copy(*out, *in)
	}
	in.Testing.DeepCopyInto(&out.Testing)
	out.Sidecars = in.Sidecars
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProvidersConfig.
func (in *ProvidersConfig) DeepCopy() *ProvidersConfig {
	if in == nil {
		return nil
	}
	out := new(ProvidersConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PublicWebService) DeepCopyInto(out *PublicWebService) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PublicWebService.
func (in *PublicWebService) DeepCopy() *PublicWebService {
	if in == nil {
		return nil
	}
	out := new(PublicWebService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceConfig) DeepCopyInto(out *ServiceConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceConfig.
func (in *ServiceConfig) DeepCopy() *ServiceConfig {
	if in == nil {
		return nil
	}
	out := new(ServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceMeshConfig) DeepCopyInto(out *ServiceMeshConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceMeshConfig.
func (in *ServiceMeshConfig) DeepCopy() *ServiceMeshConfig {
	if in == nil {
		return nil
	}
	out := new(ServiceMeshConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Sidecar) DeepCopyInto(out *Sidecar) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Sidecar.
func (in *Sidecar) DeepCopy() *Sidecar {
	if in == nil {
		return nil
	}
	out := new(Sidecar)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Sidecars) DeepCopyInto(out *Sidecars) {
	*out = *in
	out.TokenRefresher = in.TokenRefresher
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Sidecars.
func (in *Sidecars) DeepCopy() *Sidecars {
	if in == nil {
		return nil
	}
	out := new(Sidecars)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestingConfig) DeepCopyInto(out *TestingConfig) {
	*out = *in
	in.Iqe.DeepCopyInto(&out.Iqe)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestingConfig.
func (in *TestingConfig) DeepCopy() *TestingConfig {
	if in == nil {
		return nil
	}
	out := new(TestingConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestingSpec) DeepCopyInto(out *TestingSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestingSpec.
func (in *TestingSpec) DeepCopy() *TestingSpec {
	if in == nil {
		return nil
	}
	out := new(TestingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenRefresherConfig) DeepCopyInto(out *TokenRefresherConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenRefresherConfig.
func (in *TokenRefresherConfig) DeepCopy() *TokenRefresherConfig {
	if in == nil {
		return nil
	}
	out := new(TokenRefresherConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UiSpec) DeepCopyInto(out *UiSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UiSpec.
func (in *UiSpec) DeepCopy() *UiSpec {
	if in == nil {
		return nil
	}
	out := new(UiSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebConfig) DeepCopyInto(out *WebConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebConfig.
func (in *WebConfig) DeepCopy() *WebConfig {
	if in == nil {
		return nil
	}
	out := new(WebConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebServices) DeepCopyInto(out *WebServices) {
	*out = *in
	out.Public = in.Public
	out.Private = in.Private
	out.Metrics = in.Metrics
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebServices.
func (in *WebServices) DeepCopy() *WebServices {
	if in == nil {
		return nil
	}
	out := new(WebServices)
	in.DeepCopyInto(out)
	return out
}
