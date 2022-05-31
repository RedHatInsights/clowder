package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestKafkaReconcilerFns(t *testing.T) {

}

func TestConverterFuncs(t *testing.T) {
	t.Run("Test intMin", func(t *testing.T) {
		answer, _ := utils.IntMin([]string{"4", "6", "7"})
		if answer != "4" {
			t.Errorf("Min function should have returned 4, returned %s", answer)
		}
	})
	t.Run("Test intMax", func(t *testing.T) {
		answer, _ := utils.IntMax([]string{"4", "6", "7"})
		if answer != "7" {
			t.Errorf("Min function should have returned 7, returned %s", answer)
		}
	})
	t.Run("Test ListMerge", func(t *testing.T) {
		answer, _ := utils.ListMerge([]string{"4,5,6", "6", "7,2"})
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
			Deployments: []crd.Deployment{{
				PodSpec: crd.PodSpec{
					Image: "test:test",
				},
				Name: "testpod",
			}},
			EnvName: "testing-env",
		},
	}
	utils.MakeService(&svc, nn, labels, ports, baseResource, false)
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
			Deployments: []crd.Deployment{{
				PodSpec: crd.PodSpec{
					Image: "test:test",
				},
				Name: "testpod",
			}},
			EnvName: "testing-env",
		},
	}
	size := resource.MustParse("1Gi")

	utils.MakePVC(&pvc, nn, labels, "1Gi", baseResource)
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
			Deployments: []crd.Deployment{{
				PodSpec: crd.PodSpec{
					Image: "test:test",
				},
				Name: "testpod",
			}},
			EnvName: "testing-env",
		},
	}

	labeler := utils.GetCustomLabeler(labels, nn, baseResource)

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
	decodedValue, _ := utils.B64Decode(&s, "key")
	assert.Equal(t, decodedValue, "number", "should decode the right value")
}

func TestRandString(t *testing.T) {
	a := utils.RandString(12)
	b := utils.RandString(12)
	assert.NotEqual(t, a, b)
}

func TestPodAnnotationsUpdate(t *testing.T) {
	table := []struct {
		name        string
		podTemplate core.PodTemplateSpec
		labels      map[string]string
		result      map[string]string
	}{
		{
			name: "test-pod-annotations-only",
			podTemplate: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test": "result",
					},
				},
			},
			labels: map[string]string{},
			result: map[string]string{
				"test": "result",
			},
		},
		{
			name: "test-pod-labels-only",
			podTemplate: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
			},
			labels: map[string]string{
				"test2": "result2",
			},
			result: map[string]string{
				"test2": "result2",
			},
		},
		{
			name: "test-pod-labels-and annotations",
			podTemplate: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test": "result",
					},
				},
			},
			labels: map[string]string{
				"test2": "result2",
			},
			result: map[string]string{
				"test":  "result",
				"test2": "result2",
			},
		},
		{
			name: "test-pod-labels-and annotations",
			podTemplate: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
			},
			labels: map[string]string{},
			result: map[string]string{},
		},
	}

	for i := range table {
		i := i
		t.Run(table[i].name, func(t *testing.T) {
			utils.UpdatePodTemplateAnnotations(&table[i].podTemplate, table[i].labels)
			assert.Equal(t, table[i].result, table[i].podTemplate.GetAnnotations(), "labels don't match")
		})
	}
}
