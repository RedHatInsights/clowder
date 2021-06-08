package serviceaccount

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func createServiceAccountForClowdObj(cache *providers.ObjectCache, ident providers.ResourceIdent, obj object.ClowdObject) error {

	if obj.GetClowdNamespace() == "" {
		err := errors.New("targetNamespace not yet populated")
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

func CreateServiceAccount(cache *providers.ObjectCache, ident providers.ResourceIdent, nn types.NamespacedName, labeler func(v1.Object)) error {

	sa := &core.ServiceAccount{}

	if err := cache.Create(ident, nn, sa); err != nil {
		return err
	}

	labeler(sa)

	if err := cache.Update(ident, sa); err != nil {
		return err
	}

	return nil
}

func CreateRoleBinding(cache *providers.ObjectCache, ident providers.ResourceIdent, nn types.NamespacedName, labeler func(v1.Object), accessLevel crd.K8sAccessLevel) error {
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

	if err := cache.Update(ident, rb); err != nil {
		return err
	}

	return nil
}
