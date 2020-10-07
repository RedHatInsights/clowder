package providers

import (
	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getRedisTestEnv() crd.ClowdEnvironment {
	return crd.ClowdEnvironment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "env",
		},
		Spec: crd.ClowdEnvironmentSpec{
			Kafka: crd.KafkaConfig{
				Provider: "local",
			},
			InMemoryDB: crd.InMemoryDBConfig{
				Provider: "redis",
			},
		},
	}
}

// func TestLocalRedis(t *testing.T) {
// 	env := getRedisTestEnv()

// 	dd, svc, pvc := apps.Deployment{}, core.Service{}, core.PersistentVolumeClaim{}
// 	makeLocalRedis(&env, &dd, &svc, &pvc)

// 	if dd.GetName() != "env-redis" {
// 		t.Errorf("Name was not set correctly, got: %v, want: %v", dd.GetName(), "env-redis")
// 	}
// 	if len(svc.Spec.Ports) < 1 {
// 		t.Errorf("Number of ports specified is wrong")
// 	}

// 	p := svc.Spec.Ports[0]
// 	if p.Port != 6379 {
// 		t.Errorf("Port number is incorrect, got: %v, want: %v", p.Port, 6379)
// 	}
// }
