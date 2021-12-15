package object

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClowdObject is used to be able to treat ClowdEnv and ClowdApp as the same type
type ClowdObject interface {
	MakeOwnerReference() metav1.OwnerReference
	GetLabels() map[string]string
	GetClowdNamespace() string
	GetClowdName() string
	GetUID() types.UID
	GetClowdSAName() string
	GetPrimaryLabel() string
	GroupVersionKind() schema.GroupVersionKind
	GetNamespacesInEnv(context.Context, client.Client) ([]string, error)
}

// LabeledClowdObject is used to be able to treat ClowdEnv and ClowdApp as the same type
type LabeledClowdObject interface {
	MakeOwnerReference() metav1.OwnerReference
	GetLabels() map[string]string
}
