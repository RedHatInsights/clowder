package object

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClowdObject interface {
	MakeOwnerReference() metav1.OwnerReference
	GetLabels() map[string]string
	GetClowdNamespace() string
	GetClowdName() string
}
