package makers

import (
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	nn = types.NamespacedName{
		Name:      "test",
		Namespace: "testNS",
	}
)

func TestMakeDeployment(t *testing.T) {
	dd := apps.Deployment{}
	pp := &crd.Application{}
	makeRedisDeployment(&dd, nn, pp)

	if dd.GetName() != nn.Name {
		t.Errorf("Name was not set correctly, got: %v, want: %v", dd.GetName(), nn.Name)
	}
}

func TestMakeService(t *testing.T) {
	s := core.Service{}
	pp := &crd.Application{}
	makeRedisService(&s, nn, pp)

	if len(s.Spec.Ports) < 1 {
		t.Errorf("Number of ports specified is wrong")
	}

	p := s.Spec.Ports[0]
	if p.Port != 5432 {
		t.Errorf("Port number is incorrect, got: %v, want: %v", p.Port, 5432)
	}
}
