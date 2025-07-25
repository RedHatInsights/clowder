package serviceaccount

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

func createServiceAccountForClowdObj(cache *rc.ObjectCache, ident rc.ResourceIdent, obj object.ClowdObject) error {

	if obj.GetClowdNamespace() == "" {
		err := errors.NewClowderError("targetNamespace not yet populated")
		err.Requeue = true
		return err
	}

	nn := types.NamespacedName{
		Name:      obj.GetClowdSAName(),
		Namespace: obj.GetClowdNamespace(),
	}

	labeler := utils.GetCustomLabeler(nil, nn, obj)

	return CreateServiceAccount(cache, ident, nn, labeler)
}

func CreateServiceAccount(cache *rc.ObjectCache, ident rc.ResourceIdent, nn types.NamespacedName, labeler func(v1.Object)) error {

	sa := &core.ServiceAccount{}

	if err := cache.Create(ident, nn, sa); err != nil {
		return err
	}

	labeler(sa)

	return cache.Update(ident, sa)
}

func CreateRoleBinding(cache *rc.ObjectCache, ident rc.ResourceIdent, nn types.NamespacedName, labeler func(v1.Object), accessLevel crd.K8sAccessLevel) error {
	if accessLevel == "default" || accessLevel == "" {
		return nil
	}

	rb := &rbac.RoleBinding{}

	if err := cache.Create(ident, nn, rb); err != nil {
		return err
	}

	labeler(rb)

	rb.Subjects = []rbac.Subject{{
		Kind:      "ServiceAccount",
		Name:      nn.Name,
		Namespace: nn.Namespace,
	}}
	rb.RoleRef = rbac.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
	}

	switch accessLevel {
	case "view":
		rb.RoleRef.Name = "view"
	case "edit":
		rb.RoleRef.Name = "edit"
	}

	return cache.Update(ident, rb)
}
