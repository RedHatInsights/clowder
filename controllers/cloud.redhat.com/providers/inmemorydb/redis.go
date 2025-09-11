package inmemorydb

import (
	"fmt"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	providerUtils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// RedisSecret is the ident referring to the redis creds secret object.
// This is needed for allowing shared redis/inmemorydb instances
var RedisSecret = rc.NewSingleResourceIdent(ProvName, "redis_secret", &core.Secret{})

type localRedis struct {
	providers.Provider
}

// NewLocalRedis returns a new local redis provider object.
func NewLocalRedis(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		RedisDeployment,
		RedisService,
		RedisConfigMap,
		RedisSecret,
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

	providerUtils.DebugLog(r.Log, "sharedinMemoryDbAppName", "app", app.Name, "sharedInMemoryDbAppName", app.Spec.SharedInMemoryDBAppName)

	if app.Spec.SharedInMemoryDBAppName != "" {
		return r.processSharedInMemoryDb(app)
	}

	sslmode := false

	creds := config.InMemoryDBConfig{}

	nn := providers.GetNamespacedName(app, "redis")

	dataInit := func() map[string]string {

		hostname := fmt.Sprintf("%v-redis.%v.svc", app.Name, app.Namespace)
		port := "6379"
		sslmode := fmt.Sprintf("%t", sslmode)

		return map[string]string{
			"hostname": hostname,
			"port":     port,
			"sslmode":  sslmode,
		}
	}

	secMap, err := providers.MakeOrGetSecret(app, r.Cache, RedisSecret, nn, dataInit)
	if err != nil {
		return errors.Wrap("Couldn't set/get secret", err)
	}

	if err = creds.Populate(secMap); err != nil {
		return errors.Wrap("couldn't populate creds", err)
	}

	configMap := &core.ConfigMap{}

	err = r.Cache.Create(RedisConfigMap, nn, configMap)

	if err != nil {
		return err
	}

	labeler := utils.MakeLabeler(nn, nil, app)
	labeler(configMap)

	configMap.Data = map[string]string{"redis.conf": "stop-writes-on-bgsave-error no\nprotected-mode no"}

	err = r.Cache.Update(RedisConfigMap, configMap)

	if err != nil {
		return err
	}

	r.Config.InMemoryDb = &creds

	objList := []rc.ResourceIdent{
		RedisDeployment,
		RedisService,
	}

	return providers.CachedMakeComponent(r, objList, app, "redis", makeLocalRedis, false)
}

func makeLocalRedis(env *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) error {
	nn := providers.GetNamespacedName(o, "redis")

	dd := objMap[RedisDeployment].(*apps.Deployment)
	svc := objMap[RedisService].(*core.Service)

	oneReplica := int32(1)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labels["service"] = "redis"
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Labels = labels
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

func (r *localRedis) processSharedInMemoryDb(app *crd.ClowdApp) error {
	providerUtils.DebugLog(r.Log, "in processSharedInMemoryDb", "app", app.Name)

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

	providerUtils.DebugLog(r.Log, "found inMemoryDb ref app", "app", app.Name, "refApp", refApp.Name)

	inn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-redis", refApp.Name),
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
