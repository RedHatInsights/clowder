package object

import (
	"cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ClowdObject is used to be able to treat ClowdEnv and ClowdApp as the same type
type ClowdObject interface {
	MakeOwnerReference() metav1.OwnerReference
	GetLabels() map[string]string
	GetClowdNamespace() string
	GetClowdName() string
	GetUID() types.UID
	GetDeploymentStatus() *common.DeploymentStatus
	GetClowdSAName() string
	GetPrimaryLabel() string
}
