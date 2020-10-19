package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const webPort = 8000

func TestKafkaReconcilerFns(t *testing.T) {

}

func TestConverterFuncs(t *testing.T) {
	t.Run("Test intMin", func(t *testing.T) {
		answer, _ := IntMin([]string{"4", "6", "7"})
		if answer != "4" {
			t.Errorf("Min function should have returned 4, returned %s", answer)
		}
	})
	t.Run("Test intMax", func(t *testing.T) {
		answer, _ := IntMax([]string{"4", "6", "7"})
		if answer != "7" {
			t.Errorf("Min function should have returned 7, returned %s", answer)
		}
	})
	t.Run("Test ListMerge", func(t *testing.T) {
		answer, _ := ListMerge([]string{"4,5,6", "6", "7,2"})
		if answer != "2,4,5,6,7" {
			t.Errorf("Min function should have returned 2,4,5,6,7 returned %s", answer)
		}
	})
}

func TestMakeService(t *testing.T) {
	svc := core.Service{}
	nn := types.NamespacedName{
		Name:      "reqapp",
		Namespace: "reqnamespace",
	}
	labels := map[string]string{
		"customLabel": "5",
	}
	ports := []core.ServicePort{
		{
			Name:     "web",
			Protocol: "TCP",
			Port:     9000,
		},
		{
			Name:     "db",
			Protocol: "TCP",
			Port:     5432,
		},
	}

	objMeta := metav1.ObjectMeta{
		Name:      nn.Name,
		Namespace: nn.Namespace,
	}

	baseResource := &crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Pods: []crd.PodSpec{{
				Image: "test:test",
				Name:  "testpod",
			}},
			EnvName: "testing-env",
		},
	}
	MakeService(&svc, nn, labels, ports, baseResource)
	assert.Equal(t, svc.Labels["app"], baseResource.Name, "should have app label")
	assert.Equal(t, svc.Labels["customLabel"], "5", "should have custom label")
	assert.Equal(t, svc.Spec.Ports[0], ports[0], "ports should be equal")
	assert.Equal(t, svc.Spec.Ports[1], ports[1], "ports should be equal")
}

func TestMakePVC(t *testing.T) {
	pvc := core.PersistentVolumeClaim{}
	nn := types.NamespacedName{
		Name:      "reqapp",
		Namespace: "reqnamespace",
	}
	labels := map[string]string{
		"customLabel": "5",
	}

	objMeta := metav1.ObjectMeta{
		Name:      nn.Name,
		Namespace: nn.Namespace,
	}

	baseResource := &crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Pods: []crd.PodSpec{{
				Image: "test:test",
				Name:  "testpod",
			}},
			EnvName: "testing-env",
		},
	}
	size := resource.MustParse("1Gi")

	MakePVC(&pvc, nn, labels, "1Gi", baseResource)
	assert.Equal(t, pvc.Labels["app"], baseResource.Name, "should have app label")
	assert.Equal(t, pvc.Labels["customLabel"], "5", "should have custom label")
	assert.Equal(t, pvc.Spec.Resources.Requests["storage"], size, "should have size set correctly")
	assert.Equal(t, pvc.Spec.AccessModes[0], core.ReadWriteOnce, "should have rwo set")
}

func TestGetCustomLabeler(t *testing.T) {

	labels := map[string]string{
		"customLabel": "5",
	}
	nn := types.NamespacedName{
		Name:      "reqapp",
		Namespace: "reqnamespace",
	}

	objMeta := metav1.ObjectMeta{
		Name:      nn.Name,
		Namespace: nn.Namespace,
	}

	baseResource := &crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Pods: []crd.PodSpec{{
				Image: "test:test",
				Name:  "testpod",
			}},
			EnvName: "testing-env",
		},
	}

	labeler := getCustomLabeler(labels, nn, baseResource)

	pvc := &core.PersistentVolumeClaim{}

	labeler(pvc)

	assert.Equal(t, pvc.Labels["app"], baseResource.Name, "should have app label")
	assert.Equal(t, pvc.Labels["customLabel"], "5", "should have custom label")
}

func TestBase64Decode(t *testing.T) {
	s := core.Secret{
		Data: map[string][]byte{
			"key": []byte("bnVtYmVy"),
		},
	}
	decodedValue, _ := B64Decode(&s, "key")
	assert.Equal(t, decodedValue, "number", "should decode the right value")
}

func TestRandString(t *testing.T) {
	a := RandString(12)
	b := RandString(12)
	assert.NotEqual(t, a, b)
}
