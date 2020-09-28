package providers

import (
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTestEnv() crd.ClowdEnvironment {
	return crd.ClowdEnvironment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "env",
			Namespace: "test",
		},
		Spec: crd.ClowdEnvironmentSpec{
			Kafka: crd.KafkaConfig{
				Provider: "local",
			},
		},
	}
}

func TestLocalKafka(t *testing.T) {
	env := getTestEnv()

	dd, svc, pvc := apps.Deployment{}, core.Service{}, core.PersistentVolumeClaim{}

	makeLocalKafka(&env, &dd, &svc, &pvc)

	if dd.Name != "env-kafka" {
		t.Errorf("Wrong deployment name %s; expected %s", dd.Name, "env-kafka")
	}
}

func TestLocalZookeeper(t *testing.T) {
	env := getTestEnv()

	dd, svc, pvc := apps.Deployment{}, core.Service{}, core.PersistentVolumeClaim{}

	makeLocalZookeeper(&env, &dd, &svc, &pvc)

	if dd.Name != "env-zookeeper" {
		t.Errorf("Wrong deployment name %s; expected %s", dd.Name, "env-kafka")
	}
}
