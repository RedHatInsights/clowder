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
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type InitContainer struct {
	Args []string `json:"args"`
}

type InsightsDatabaseSpec struct {
	Version *int32 `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
}

type ApplicationSpec struct {
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

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

func (i *Application) GetLabels() map[string]string {
	return map[string]string{"app": i.ObjectMeta.Name}
}

// GetNamespacedName contructs a new namespaced name for an object from the pattern
func (i *Application) GetNamespacedName(pattern string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: i.Namespace,
		Name:      fmt.Sprintf(pattern, i.Name),
	}
}

func (i *Application) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: i.APIVersion,
		Kind:       i.Kind,
		Name:       i.ObjectMeta.Name,
		UID:        i.ObjectMeta.UID,
	}
}

type omfunc func(o metav1.Object)

func (i *Application) SetObjectMeta(o metav1.Object, opts ...omfunc) {
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
