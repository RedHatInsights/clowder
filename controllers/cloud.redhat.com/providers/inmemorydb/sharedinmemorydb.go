package inmemorydb

import (
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	providerUtils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// SharedInMemoryDbDeployment identifies the main redis deployment
var SharedInMemoryDbDeployment = rc.NewSingleResourceIdent(ProvName, "redis_deployment", &apps.Deployment{})

// SharedInMemoryDbService identifies the main redis service
var SharedInMemoryDbService = rc.NewSingleResourceIdent(ProvName, "redis_service", &core.Service{})

// SharedInMemoryDbConfigMap identifies the main redis configmap
var SharedInMemoryDbConfigMap = rc.NewSingleResourceIdent(ProvName, "redis_config_map", &core.ConfigMap{})

type SharedInMemoryDb struct {
	providers.Provider
}

// NewSharedInMemoryDb returns a new local redis provider object.
func NewSharedInMemoryDb(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		SharedInMemoryDbDeployment,
		SharedInMemoryDbService,
		SharedInMemoryDbConfigMap,
	)
	return &SharedInMemoryDb{Provider: *p}, nil
}

func (r *SharedInMemoryDb) EnvProvide() error {
	return nil
}

func (r *SharedInMemoryDb) Provide(app *crd.ClowdApp) error {
	if !app.Spec.InMemoryDB {
		return nil
	}

	if app.Spec.SharedInMemoryDbAppName == "" {
		return nil
	}

	return r.processSharedInMemoryDb(app)

}

func makeSharedInMemoryDb(env *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) error {
	nn := providers.GetNamespacedName(o, "redis")

	dd := objMap[SharedInMemoryDbDeployment].(*apps.Deployment)
	svc := objMap[SharedInMemoryDbService].(*core.Service)

	oneReplica := int32(1)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labels["service"] = "redis"
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
		Image: providerUtils.GetInMemoryDBImage(env),
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "redis",
			ContainerPort: 6379,
			Protocol:      core.ProtocolTCP,
		}},
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		VolumeMounts: []core.VolumeMount{{
			Name:      nn.Name,
			MountPath: "/etc/redis",
		}},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
	}}

	servicePorts := []core.ServicePort{{
		Name:       "redis",
		Port:       6379,
		Protocol:   core.ProtocolTCP,
		TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 6379},
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)
	return nil
}

func (r *SharedInMemoryDb) processSharedInMemoryDb(app *crd.ClowdApp) error {
	err := checkDependency(app)

	if err != nil {
		return err
	}

	shimdbCfg := config.InMemoryDBConfig{}

	refApp, err := crd.GetAppForDBInSameEnv(r.Ctx, r.Client, app, true)

	if err != nil {
		return err
	}

	secret := core.Secret{}

	// **TODO** This won't reference the right secret currently, need to point it to the redis service access info
	inn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-db", refApp.Name),
		Namespace: refApp.Namespace,
	}

	// This is a REAL call here, not a cached call as the reconciliation must have been processed
	// for the app we depend on.
	if err = r.Client.Get(r.Ctx, inn, &secret); err != nil {
		return errors.Wrap("Couldn't set/get secret", err)
	}

	secMap := make(map[string]string)

	for k, v := range secret.Data {
		(secMap)[k] = string(v)
	}

	err = shimdbCfg.Populate(&secMap)
	if err != nil {
		return errors.Wrap("couldn't convert to int", err)
	}

	r.Config.InMemoryDb = &shimdbCfg

	return nil
}
