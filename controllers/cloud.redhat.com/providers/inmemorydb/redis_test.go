package inmemorydb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
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
	_ = makeLocalRedis(&env, &env, objMap, true, false)

	assert.Equal(t, "env-redis", dd.GetName(), "name was not set correctly")
	assert.Len(t, svc.Spec.Ports, 1, "number of ports specified is wrong")
	assert.Equal(t, int32(6379), svc.Spec.Ports[0].Port, "port number is incorrect")
}

func TestLocalRedisImageOverride(t *testing.T) {
	env := getRedisTestEnv()
	env.Spec.Providers.InMemoryDB.Image = "testing.com/test/image"

	dd, svc := apps.Deployment{}, core.Service{}
	objMap := providers.ObjectMap{
		RedisDeployment: &dd,
		RedisService:    &svc,
	}
	_ = makeLocalRedis(&env, &env, objMap, true, false)

	assert.Equal(t, "env-redis", dd.GetName(), "name was not set correctly")
	assert.Len(t, svc.Spec.Ports, 1, "number of ports specified is wrong")
	assert.Equal(t, int32(6379), svc.Spec.Ports[0].Port, "port number is incorrect")
	assert.Equal(t, "testing.com/test/image", dd.Spec.Template.Spec.Containers[0].Image)
}
