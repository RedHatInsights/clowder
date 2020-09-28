package providers

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type redisProvider struct {
	Provider
	Config config.InMemoryDBConfig
}

func (r *redisProvider) Configure(config *config.AppConfig) {
	config.InMemoryDb = &r.Config
}

func makeRedisDeployment(dd *apps.Deployment, nn types.NamespacedName, pp *crd.ClowdApp) {
	oneReplica := int32(1)

	pp.SetObjectMeta(
		dd,
		crd.Name(nn.Name),
		crd.Namespace(nn.Namespace),
	)
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: pp.GetLabels()}
	dd.Spec.Template.ObjectMeta.Labels = pp.GetLabels()
	dd.Spec.Replicas = &oneReplica

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: "redis:6",
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "redis",
			ContainerPort: 6379,
		}},
	}}
}

func makeRedisService(s *core.Service, nn types.NamespacedName, pp *crd.ClowdApp) {
	servicePorts := []core.ServicePort{{
		Name:     "redis",
		Port:     6379,
		Protocol: "TCP",
	}}

	pp.SetObjectMeta(
		s,
		crd.Name(nn.Name),
		crd.Namespace(nn.Namespace),
	)

	s.Spec.Selector = pp.GetLabels()
	s.Spec.Ports = servicePorts
}

func (r *redisProvider) CreateInMemoryDB(app *crd.ClowdApp) error {
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%v-redis", app.Name),
		Namespace: app.Namespace,
	}

	dd := apps.Deployment{}

	update, err := utils.UpdateOrErr(r.Client.Get(r.Ctx, nn, &dd))

	if err != nil {
		return err
	}

	makeRedisDeployment(&dd, nn, app)

	if _, err = update.Apply(r.Ctx, r.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	update, err = utils.UpdateOrErr(r.Client.Get(r.Ctx, nn, &s))

	if err != nil {
		return err
	}

	makeRedisService(&s, nn, app)

	if _, err = update.Apply(r.Ctx, r.Client, &s); err != nil {
		return err
	}

	r.Config = config.InMemoryDBConfig{
		Hostname: fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
		Port:     6379,
	}

	return nil
}

func NewRedis(p *Provider) (InMemoryDBProvider, error) {
	return &redisProvider{Provider: *p}, nil
}
