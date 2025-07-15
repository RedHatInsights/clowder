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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// ClowdAppRefDeployment defines a deployment reference for a service on a different cluster
type ClowdAppRefDeployment struct {
	// Name defines the identifier of the deployment on the remote cluster
	Name string `json:"name"`

	// Hostname defines the hostname where the service can be reached
	Hostname string `json:"hostname"`

	// Port defines the port where the service can be reached (defaults to 8000)
	Port int32 `json:"port,omitempty"`

	// TlsPort defines the TLS port where the service can be reached (defaults to 8443)
	TlsPort int32 `json:"tlsPort,omitempty"`

	// PrivatePort defines the private port where the service can be reached (defaults to 10000)
	PrivatePort int32 `json:"privatePort,omitempty"`

	// TlsPrivatePort defines the TLS private port where the service can be reached (defaults to 10443)
	TlsPrivatePort int32 `json:"tlsPrivatePort,omitempty"`

	// Web indicates if this deployment has a public web service
	Web bool `json:"web,omitempty"`

	// WebServices defines the web services configuration for this deployment
	WebServices ClowdAppRefWebServices `json:"webServices,omitempty"`

	// APIPath defines the API path for this deployment (deprecated, use APIPaths)
	APIPath string `json:"apiPath,omitempty"`

	// APIPaths defines the API paths for this deployment
	APIPaths []string `json:"apiPaths,omitempty"`
}

// ClowdAppRefWebServices defines the web services configuration for a ClowdAppRef deployment
type ClowdAppRefWebServices struct {
	// Public defines the public web service configuration
	Public ClowdAppRefPublicWebService `json:"public,omitempty"`

	// Private defines the private web service configuration
	Private ClowdAppRefPrivateWebService `json:"private,omitempty"`
}

// ClowdAppRefPublicWebService defines the public web service configuration for a ClowdAppRef deployment
type ClowdAppRefPublicWebService struct {
	// Enabled indicates if the public web service is enabled
	Enabled bool `json:"enabled,omitempty"`
}

// ClowdAppRefPrivateWebService defines the private web service configuration for a ClowdAppRef deployment
type ClowdAppRefPrivateWebService struct {
	// Enabled indicates if the private web service is enabled
	Enabled bool `json:"enabled,omitempty"`
}

// ClowdAppRefSpec defines the desired state of ClowdAppRef
type ClowdAppRefSpec struct {
	// The name of the ClowdEnvironment resource that this ClowdAppRef will be used in
	EnvName string `json:"envName"`

	// A list of deployments that represent services on a different cluster
	Deployments []ClowdAppRefDeployment `json:"deployments"`

	// RemoteCluster defines information about the remote cluster where the services are located
	RemoteCluster ClowdAppRefRemoteCluster `json:"remoteCluster,omitempty"`

	// Disabled turns off this ClowdAppRef
	Disabled bool `json:"disabled,omitempty"`
}

// ClowdAppRefRemoteCluster defines information about the remote cluster
type ClowdAppRefRemoteCluster struct {
	// Name defines the name of the remote cluster
	Name string `json:"name,omitempty"`

	// Region defines the region of the remote cluster
	Region string `json:"region,omitempty"`

	// Environment defines the environment of the remote cluster (e.g., prod, stage)
	Environment string `json:"environment,omitempty"`
}

// ClowdAppRefStatus defines the observed state of ClowdAppRef
type ClowdAppRefStatus struct {
	// Ready indicates if the ClowdAppRef is ready to be used
	Ready bool `json:"ready"`

	// Conditions represents the latest available observations of the ClowdAppRef's current state
	Conditions []clusterv1.Condition `json:"conditions,omitempty"`
}

// ClowdAppRef is the Schema for the clowdapprefs API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Env",type="string",JSONPath=".spec.envName"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ClowdAppRef struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClowdAppRefSpec   `json:"spec,omitempty"`
	Status ClowdAppRefStatus `json:"status,omitempty"`
}

// ClowdAppRefList contains a list of ClowdAppRef
// +kubebuilder:object:root=true
type ClowdAppRefList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClowdAppRef `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClowdAppRef{}, &ClowdAppRefList{})
}

// GetConditions returns the conditions from the ClowdAppRef status
func (car *ClowdAppRef) GetConditions() clusterv1.Conditions {
	return car.Status.Conditions
}

// SetConditions sets the conditions on the ClowdAppRef status
func (car *ClowdAppRef) SetConditions(conditions clusterv1.Conditions) {
	car.Status.Conditions = conditions
}

// GetLabels returns the labels that should be applied to child resources
func (car *ClowdAppRef) GetLabels() map[string]string {
	if car.Labels == nil {
		car.Labels = map[string]string{}
	}

	if car.Labels["app"] == "" {
		car.Labels["app"] = car.Name
	}

	newMap := make(map[string]string)
	newMap["app"] = car.Labels["app"]

	return newMap
}

// GetNamespacedName returns the namespaced name for the ClowdAppRef
func (car *ClowdAppRef) GetNamespacedName(pattern string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: car.Namespace,
		Name:      car.Name,
	}
}

// GetIdent returns the identity of the ClowdAppRef
func (car *ClowdAppRef) GetIdent() string {
	return car.Name + ":" + car.Namespace
}

// MakeOwnerReference creates an owner reference for the ClowdAppRef
func (car *ClowdAppRef) MakeOwnerReference() metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: car.APIVersion,
		Kind:       car.Kind,
		Name:       car.Name,
		UID:        car.UID,
	}
}

// GetPrimaryLabel returns the primary label for the ClowdAppRef
func (car *ClowdAppRef) GetPrimaryLabel() string {
	return "clowdappref"
}

// GetClowdNamespace returns the namespace for the ClowdAppRef
func (car *ClowdAppRef) GetClowdNamespace() string {
	return car.Namespace
}

// GetClowdName returns the name for the ClowdAppRef
func (car *ClowdAppRef) GetClowdName() string {
	return car.Name
}

// GetUID returns the UID for the ClowdAppRef
func (car *ClowdAppRef) GetUID() types.UID {
	return car.UID
}

// IsReady returns true if the ClowdAppRef is ready
func (car *ClowdAppRef) IsReady() bool {
	return car.Status.Ready
}

// GetDeploymentNamespacedName returns the namespaced name for a deployment
func (car *ClowdAppRef) GetDeploymentNamespacedName(d *ClowdAppRefDeployment) types.NamespacedName {
	return types.NamespacedName{
		Name:      car.Name + "-" + d.Name,
		Namespace: car.Namespace,
	}
}

// SetObjectMeta sets the object metadata for resources created by this ClowdAppRef
func (car *ClowdAppRef) SetObjectMeta(o metav1.Object, opts ...func(metav1.Object)) {
	o.SetName(car.Name)
	o.SetNamespace(car.Namespace)
	o.SetLabels(car.GetLabels())
	o.SetOwnerReferences([]metav1.OwnerReference{car.MakeOwnerReference()})

	for _, opt := range opts {
		opt(o)
	}
}
