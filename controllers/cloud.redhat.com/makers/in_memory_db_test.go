package makers

import (
	"testing"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestMakeDeployment(t *testing.T) {
	dd := apps.Deployment{}
	nn := types.NamespacedName{
		Name:      "test",
		Namespace: "testNS",
	}
	pp := &crd.InsightsApp{}
	makeRedisDeployment(&dd, nn, pp)

	if dd.GetName() != nn.Name {
		t.Errorf("Name was not set correctly, got: %v, want: %v", dd.GetName(), nn.Name)
	}
}
