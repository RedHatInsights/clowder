package inmemorydb

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// RedisDeployment identifies the main redis deployment
var RedisDeployment = rc.NewSingleResourceIdent(ProvName, "redis_deployment", &apps.Deployment{})

// RedisService identifies the main redis service
var RedisService = rc.NewSingleResourceIdent(ProvName, "redis_service", &core.Service{})

// RedisConfigMap identifies the main redis configmap
var RedisConfigMap = rc.NewSingleResourceIdent(ProvName, "redis_config_map", &core.ConfigMap{})

type localRedis struct {
	providers.Provider
}

// NewLocalRedis returns a new local redis provider object.
func NewLocalRedis(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		RedisDeployment,
		RedisService,
		RedisConfigMap,
	)
	return &localRedis{Provider: *p}, nil
}

func (r *localRedis) EnvProvide() error {
	return nil
}

func (r *localRedis) Provide(app *crd.ClowdApp) error {
	if !app.Spec.InMemoryDB {
		return nil
	}

	sslmode := false

	creds := config.InMemoryDBConfig{}

	creds.Hostname = fmt.Sprintf("%v-redis.%v.svc", app.Name, app.Namespace)
	creds.Port = 6379
	creds.SslMode = &sslmode
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

	r.Config.InMemoryDb = &creds

	objList := []rc.ResourceIdent{
		RedisDeployment,
		RedisService,
	}

	return providers.CachedMakeComponent(r.Provider.Cache, objList, app, "redis", makeLocalRedis, false, r.Env.IsNodePort())
}

func makeLocalRedis(o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) {
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
				DefaultMode: utils.Int32Ptr(420),
				LocalObjectReference: core.LocalObjectReference{
					Name: nn.Name,
				},
			},
		}},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: DefaultImageInMemoryDBRedis,
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
