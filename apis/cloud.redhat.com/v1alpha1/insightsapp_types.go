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
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InitContainer struct {
	Args []string `json:"args"`
}

type InsightsDatabaseSpec struct {
	Version *int32 `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
}

type InsightsAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	MinReplicas    *int32                   `json:"minReplicas,omitempty"`
	Image          string                   `json:"image"`
	Command        []string                 `json:"command,omitempty"`
	Args           []string                 `json:"args,omitempty"`
	Env            []v1.EnvVar              `json:"env,omitempty"`
	Resources      v1.ResourceRequirements  `json:"resources,omitempty"`
	LivenessProbe  *v1.Probe                `json:"livenessProbe,omitempty"`
	ReadinessProbe *v1.Probe                `json:"readinessProbe,omitempty"`
	Volumes        []v1.Volume              `json:"volumes,omitempty"`
	VolumeMounts   []v1.VolumeMount         `json:"volumeMounts,omitempty"`
	Web            bool                     `json:"web,omitempty"`
	Base           string                   `json:"base"`
	KafkaTopics    []strimzi.KafkaTopicSpec `json:"kafkaTopics,omitempty"`
	Database       InsightsDatabaseSpec     `json:"database,omitempty"`
	ObjectStore    []string                 `json:"objectStore,omitempty"`
	InMemoryDB     bool                     `json:"inMemoryDb,omitempty"`
	Dependencies   []string                 `json:"dependencies,omitempty"`
}

// InsightsAppStatus defines the observed state of InsightsApp
type InsightsAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InsightsApp is the Schema for the insightsapps API
type InsightsApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InsightsAppSpec   `json:"spec,omitempty"`
	Status InsightsAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InsightsAppList contains a list of InsightsApp
type InsightsAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InsightsApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InsightsApp{}, &InsightsAppList{})
}

func (i *InsightsApp) GetLabels() map[string]string {
	return map[string]string{"app": i.ObjectMeta.Name}
}

func (i *InsightsApp) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
	}
}

type omfunc func(o metav1.Object)

func (i *InsightsApp) SetObjectMeta(o metav1.Object, opts ...omfunc) {
	o.SetName(i.Name)
	o.SetNamespace(i.Namespace)
	o.SetLabels(i.GetLabels())
	o.SetOwnerReferences([]metav1.OwnerReference{i.MakeOwnerReference()})

	for _, opt := range opts {
		opt(o)
	}
}

func Name(name string) omfunc {
	return func(o metav1.Object) {
		o.SetName(name)
	}
}

func Namespace(namespace string) omfunc {
	return func(o metav1.Object) {
		o.SetNamespace(namespace)
	}
}

func UnsetOwner() omfunc {
	return func(o metav1.Object) {
		o.SetOwnerReferences([]metav1.OwnerReference{{}})
	}
}

func Labels(labels map[string]string) omfunc {
	return func(o metav1.Object) {
		o.SetLabels(labels)
	}
}
