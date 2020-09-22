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
)

// TurnpikeAuthSpec defines the desired authn/authz policy for a route
type TurnpikeAuthSpec struct {
	Saml string `json:"saml,omitempty"`
	X509 string `json:"x509,omitempty"`
}

// TurnpikeRouteSpec defines the desired state of TurnpikeRoute
type TurnpikeRouteSpec struct {
	Route    string           `json:"route"`
	Origin   string 		  `json:"origin"`
	Auth     TurnpikeAuthSpec `json:"auth,omitempty"`
	SourceIP []string 		  `json:"source_ip,omitempty"`
}

// TurnpikeRouteStatus is the observed state of a TurnpikeRoute
type TurnpikeRouteStatus struct {
}

// +kubebuilder:object:root=true

// TurnpikeRoute is the Schema for the turnpikeroutes API
type TurnpikeRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TurnpikeRouteSpec   `json:"spec,omitempty"`
	Status TurnpikeRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TurnpikeRouteList contains a list of TurnpikeRoute
type TurnpikeRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TurnpikeRoute `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TurnpikeRoute{}, &TurnpikeRouteList{})
}
