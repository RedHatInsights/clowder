package inmemorydb

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type localRedis struct {
	p.Provider
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

	err := r.Client.Get(r.Ctx, nn, configMap)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	labeler := utils.MakeLabeler(nn, nil, app)
	labeler(configMap)

	configMap.Data = map[string]string{"redis.conf": "stop-writes-on-bgsave-error no\n"}

	err = update.Apply(r.Ctx, r.Client, configMap)
	if err != nil {
		return err
	}

	config.InMemoryDb = &r.Config

	return providers.MakeComponent(r.Ctx, r.Client, app, "redis", makeLocalRedis, r.Provider.Env.Spec.Providers.InMemoryDB.PVC)
}

// NewLocalRedis returns a new local redis provider object.
func NewLocalRedis(p *providers.Provider) (providers.ClowderProvider, error) {
	config := config.InMemoryDBConfig{}

	redisProvider := localRedis{Provider: *p, Config: config}

	return &redisProvider, nil
}

func makeLocalRedis(o obj.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim, usePVC bool) {
	nn := providers.GetNamespacedName(o, "redis")

	oneReplica := int32(1)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.ObjectMeta.Labels = labels
	dd.Spec.Replicas = &oneReplica
	dd.Spec.Template.Spec.ServiceAccountName = o.GetClowdSAName()

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
