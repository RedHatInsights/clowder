package inmemorydb

import (
	"testing"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getRedisTestEnv() crd.ClowdEnvironment {
	return crd.ClowdEnvironment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "env",
		},
		Spec: crd.ClowdEnvironmentSpec{
			Providers: crd.ProvidersConfig{
				Kafka: crd.KafkaConfig{
					Mode: "local",
				},
				InMemoryDB: crd.InMemoryDBConfig{
					Mode: "redis",
				},
			},
		},
	}
}

func TestLocalRedis(t *testing.T) {
	env := getRedisTestEnv()

	dd, svc := apps.Deployment{}, core.Service{}
	objMap := providers.ObjectMap{
		RedisDeployment: &dd,
		RedisService:    &svc,
	}
	makeLocalRedis(&env, objMap, true)

	if dd.GetName() != "env-redis" {
		t.Errorf("Name was not set correctly, got: %v, want: %v", dd.GetName(), "env-redis")
	}
	if len(svc.Spec.Ports) < 1 {
		t.Errorf("Number of ports specified is wrong")
	}

	p := svc.Spec.Ports[0]
	if p.Port != 6379 {
		t.Errorf("Port number is incorrect, got: %v, want: %v", p.Port, 6379)
	}
}
