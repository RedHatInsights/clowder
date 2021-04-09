package inmemorydb

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisDeployment identifies the main redis deployment
var RedisDeployment = providers.NewSingleResourceIdent(ProvName, "redis_deployment", &apps.Deployment{})

// RedisService identifies the main redis service
var RedisService = providers.NewSingleResourceIdent(ProvName, "redis_service", &core.Service{})

// RedisConfigMap identifies the main redis configmap
var RedisConfigMap = providers.NewSingleResourceIdent(ProvName, "redis_config_map", &core.ConfigMap{})

type localRedis struct {
	providers.Provider
	Config config.InMemoryDBConfig
}

func (r *localRedis) Provide(app *crd.ClowdApp, config *config.AppConfig) error {
	if !app.Spec.InMemoryDB {
		return nil
	}

	r.Config.Hostname = fmt.Sprintf("%v-redis.%v.svc", app.Name, app.Namespace)
	r.Config.Port = 6379

	nn := providers.GetNamespacedName(app, "redis")

	configMap := &core.ConfigMap{}

	err := r.Provider.Cache.Create(RedisConfigMap, nn, configMap)

	if err != nil {
		return err
	}

	labeler := utils.MakeLabeler(nn, nil, app)
	labeler(configMap)

	configMap.Data = map[string]string{"redis.conf": "stop-writes-on-bgsave-error no\n"}

	err = r.Provider.Cache.Update(RedisConfigMap, configMap)

	if err != nil {
		return err
	}

	config.InMemoryDb = &r.Config

	objList := []providers.ResourceIdent{
		RedisDeployment,
		RedisService,
	}

	return providers.CachedMakeComponent(r.Provider.Cache, objList, app, "redis", makeLocalRedis, false)
}

// NewLocalRedis returns a new local redis provider object.
func NewLocalRedis(p *providers.Provider) (providers.ClowderProvider, error) {
	config := config.InMemoryDBConfig{}

	redisProvider := localRedis{Provider: *p, Config: config}

	return &redisProvider, nil
}

func makeLocalRedis(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool) {
	nn := providers.GetNamespacedName(o, "redis")

	dd := objMap[RedisDeployment].(*apps.Deployment)
	svc := objMap[RedisService].(*core.Service)

	oneReplica := int32(1)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.ObjectMeta.Labels = labels
	dd.Spec.Replicas = &oneReplica

	probeHandler := core.Handler{
		Exec: &core.ExecAction{
			Command: []string{
				"redis-cli",
			},
		},
	}

	livenessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 15,
		TimeoutSeconds:      2,
	}
	readinessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 45,
		TimeoutSeconds:      2,
	}

	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: nn.Name,
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: nn.Name,
				},
			},
		}},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: "redis:6",
		Command: []string{
			"redis-server",
			"/usr/local/etc/redis/redis.conf",
		},
		Env: []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "redis",
			ContainerPort: 6379,
		}},
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		VolumeMounts: []core.VolumeMount{{
			Name:      nn.Name,
			MountPath: "/usr/local/etc/redis/",
		}},
	}}

	servicePorts := []core.ServicePort{{
		Name:     "redis",
		Port:     6379,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o)
}
