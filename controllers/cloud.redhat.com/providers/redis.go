package providers

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type localRedis struct {
	Provider
	Config config.InMemoryDBConfig
}

func (r *localRedis) Configure(config *config.AppConfig) {
	config.InMemoryDb = &r.Config
}

func NewLocalRedis(p *Provider) (InMemoryDBProvider, error) {
	return &localRedis{Provider: *p}, nil
}

func (p *localRedis) CreateInMemoryDB(app *crd.ClowdApp) error {
	config := config.InMemoryDBConfig{
		Hostname: fmt.Sprintf("%v.%v.svc", app.Name, app.Namespace),
		Port:     6379,
	}
	p.Config = config
	if err := makeComponent(&p.Provider, "redis", makeLocalRedis, app); err != nil {
		fmt.Print("-----------------------ERROR--------------------------")
		return err
	}
	return nil
}

func makeLocalRedis(app *utils.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim) {
	nn := (*app).GetNamespacedName("redis")

	oneReplica := int32(1)

	labels := (*app).GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, *app)

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

	utils.MakeService(svc, nn, nil, servicePorts, *app)
}
