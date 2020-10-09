package inmemorydb

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
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

func (r *localRedis) Configure(config *config.AppConfig) {
	config.InMemoryDb = &r.Config
}

func (r *localRedis) CreateInMemoryDB(app *crd.ClowdApp) error {
	r.Config.Hostname = fmt.Sprintf("%v-redis.%v.svc", app.Name, app.Namespace)
	r.Config.Port = 6379
	return providers.MakeComponent(r.Ctx, r.Client, app, "redis", makeLocalRedis)
}

func NewLocalRedis(p *providers.Provider) (InMemoryDBProvider, error) {
	config := config.InMemoryDBConfig{}

	redisProvider := localRedis{Provider: *p, Config: config}

	return &redisProvider, nil
}

func makeLocalRedis(o utils.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim) {
	nn := providers.GetNamespacedName(o, "redis")

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

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: "redis:6",
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "redis",
			ContainerPort: 6379,
		}},
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
	}}

	servicePorts := []core.ServicePort{{
		Name:     "redis",
		Port:     6379,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o)
}
