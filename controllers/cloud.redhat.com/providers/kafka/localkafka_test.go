package kafka

import (
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getKafkaTestEnv() crd.ClowdEnvironment {
	return crd.ClowdEnvironment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "env",
		},
		Spec: crd.ClowdEnvironmentSpec{
			Providers: crd.ProvidersConfig{
				Kafka: crd.KafkaConfig{
					Mode: "local",
				},
			},
		},
	}
}

func TestLocalKafka(t *testing.T) {
	env := getKafkaTestEnv()

	dd, svc, pvc := apps.Deployment{}, core.Service{}, core.PersistentVolumeClaim{}

	objMap := providers.ObjectMap{
		LocalKafkaDeployment: &dd,
		LocalKafkaService:    &svc,
		LocalKafkaPVC:        &pvc,
	}

	makeLocalKafka(&env, objMap, true, false)

	if dd.Name != "env-kafka" {
		t.Errorf("Wrong deployment name %s; expected %s", dd.Name, "env-kafka")
	}

	if svc.Name != "env-kafka" {
		t.Errorf("Wrong service name %s; expected %s", svc.Name, "env-kafka")
	}

	if pvc.Name != "env-kafka" {
		t.Errorf("Wrong pvc name %s; expected %s", pvc.Name, "env-kafka")
	}
}

func TestLocalZookeeper(t *testing.T) {
	env := getKafkaTestEnv()

	dd, svc, pvc := apps.Deployment{}, core.Service{}, core.PersistentVolumeClaim{}

	objMap := providers.ObjectMap{
		LocalZookeeperDeployment: &dd,
		LocalZookeeperService:    &svc,
		LocalZookeeperPVC:        &pvc,
	}

	makeLocalZookeeper(&env, objMap, true, false)

	if dd.Name != "env-zookeeper" {
		t.Errorf("Wrong deployment name %s; expected %s", dd.Name, "env-zookeeper")
	}

	if svc.Name != "env-zookeeper" {
		t.Errorf("Wrong service name %s; expected %s", svc.Name, "env-zookeeper")
	}

	if pvc.Name != "env-zookeeper" {
		t.Errorf("Wrong pvc name %s; expected %s", pvc.Name, "env-zookeeper")
	}
}
