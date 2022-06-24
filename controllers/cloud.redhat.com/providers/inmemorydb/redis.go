package inmemorydb

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

// RedisDeployment identifies the main redis deployment
var RedisDeployment = rc.NewSingleResourceIdent(ProvName, "redis_deployment", &apps.Deployment{})

// RedisService identifies the main redis service
var RedisService = rc.NewSingleResourceIdent(ProvName, "redis_service", &core.Service{})

// RedisConfigMap identifies the main redis configmap
var RedisConfigMap = rc.NewSingleResourceIdent(ProvName, "redis_config_map", &core.ConfigMap{})

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

	objList := []rc.ResourceIdent{
		RedisDeployment,
		RedisService,
	}

	return providers.CachedMakeComponent(r.Provider.Cache, objList, app, "redis", makeLocalRedis, false, r.Env.IsNodePort())
}

// NewLocalRedis returns a new local redis provider object.
func NewLocalRedis(p *providers.Provider) (providers.ClowderProvider, error) {
	config := config.InMemoryDBConfig{}

	redisProvider := localRedis{Provider: *p, Config: config}

	return &redisProvider, nil
}

func makeLocalRedis(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
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

	probeHandler := core.ProbeHandler{
		Exec: &core.ExecAction{
			Command: []string{
				"redis-cli",
			},
		},
	}

	livenessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 15,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 45,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: nn.Name,
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				DefaultMode: common.Int32Ptr(420),
				LocalObjectReference: core.LocalObjectReference{
					Name: nn.Name,
				},
			},
		}},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: IMAGE_INMEMORYDB_REDIS,
		Command: []string{
			"redis-server",
			"/usr/local/etc/redis/redis.conf",
		},
		Env: []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "redis",
			ContainerPort: 6379,
			Protocol:      core.ProtocolTCP,
		}},
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		VolumeMounts: []core.VolumeMount{{
			Name:      nn.Name,
			MountPath: "/usr/local/etc/redis/",
		}},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
	}}

	servicePorts := []core.ServicePort{{
		Name:       "redis",
		Port:       6379,
		Protocol:   core.ProtocolTCP,
		TargetPort: intstr.FromInt(int(6379)),
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)
}
