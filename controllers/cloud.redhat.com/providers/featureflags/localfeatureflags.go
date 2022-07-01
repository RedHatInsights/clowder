package featureflags

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

// LocalFFDeployment is the ident referring to the local Feature Flags deployment object.
var LocalFFDeployment = rc.NewSingleResourceIdent(ProvName, "ff_deployment", &apps.Deployment{})

// LocalFFService is the ident referring to the local Feature Flags service object.
var LocalFFService = rc.NewSingleResourceIdent(ProvName, "ff_service", &core.Service{})

// LocalFFDBDeployment is the ident referring to the local Feature Flags DB deployment object.
var LocalFFDBDeployment = rc.NewSingleResourceIdent(ProvName, "ff_db_deployment", &apps.Deployment{})

// LocalFFDBService is the ident referring to the local Feature Flags DB service object.
var LocalFFDBService = rc.NewSingleResourceIdent(ProvName, "ff_db_service", &core.Service{})

// LocalFFDBPVC is the ident referring to the local Feature Flags DB PVC object.
var LocalFFDBPVC = rc.NewSingleResourceIdent(ProvName, "ff_db_pvc", &core.PersistentVolumeClaim{})

// LocalFFDBSecret is the ident referring to the local Feature Flags DB secret object.
var LocalFFDBSecret = rc.NewSingleResourceIdent(ProvName, "ff_db_secret", &core.Secret{})

type localFeatureFlagsProvider struct {
	providers.Provider
	Config config.FeatureFlagsConfig
}

// NewLocalFeatureFlagsProvider returns a new local featureflags provider object.
func NewLocalFeatureFlagsProvider(p *providers.Provider) (providers.ClowderProvider, error) {

	ffp := &localFeatureFlagsProvider{Provider: *p, Config: config.FeatureFlagsConfig{}}

	objList := []rc.ResourceIdent{
		LocalFFDeployment,
		LocalFFService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "featureflags", makeLocalFeatureFlags, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	nn := types.NamespacedName{
		Name:      "featureflags-db",
		Namespace: p.Env.Status.TargetNamespace,
	}

	dd := &apps.Deployment{}
	if err := p.Cache.Create(LocalFFDBDeployment, nn, dd); err != nil {
		return nil, err
	}

	dbCfg := config.DatabaseConfig{}
	dataInit := func() map[string]string {
		username := utils.RandString(16)
		password := utils.RandString(16)
		pgPass := utils.RandString(16)
		hostname := fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace)
		connectionURL := fmt.Sprintf("postgres://%s:%s@%s/%s", username, password, hostname, "unleash")

		return map[string]string{
			"hostname":      hostname,
			"port":          "5432",
			"username":      username,
			"password":      password,
			"pgPass":        pgPass,
			"name":          "unleash",
			"connectionURL": connectionURL,
		}
	}

	secMap, err := providers.MakeOrGetSecret(p.Ctx, p.Env, p.Cache, LocalFFDBSecret, nn, dataInit)
	if err != nil {
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}

	dbCfg.Populate(secMap)
	dbCfg.AdminUsername = "postgres"

	ffp.Config = config.FeatureFlagsConfig{
		Hostname: fmt.Sprintf("%s-featureflags.%s.svc", p.Env.Name, p.Env.Status.TargetNamespace),
		Port:     4242,
		Scheme:   config.FeatureFlagsConfigSchemeHttp,
	}
	labels := &map[string]string{"sub": "feature_flags"}

	res := core.ResourceRequirements{
		Limits: core.ResourceList{
			"memory": resource.MustParse("200Mi"),
			"cpu":    resource.MustParse("100m"),
		},
		Requests: core.ResourceList{
			"memory": resource.MustParse("100Mi"),
			"cpu":    resource.MustParse("50m"),
		},
	}

	provutils.MakeLocalDB(dd, nn, p.Env, labels, &dbCfg, "quay.io/cloudservices/postgresql-rds:12-9ee2984", p.Env.Spec.Providers.FeatureFlags.PVC, "unleash", &res)

	if err = p.Cache.Update(LocalFFDBDeployment, dd); err != nil {
		return nil, err
	}

	s := &core.Service{}
	if err := p.Cache.Create(LocalFFDBService, nn, s); err != nil {
		return nil, err
	}

	provutils.MakeLocalDBService(s, nn, p.Env, labels)

	if err = p.Cache.Update(LocalFFDBService, s); err != nil {
		return nil, err
	}

	if p.Env.Spec.Providers.FeatureFlags.PVC {
		pvc := &core.PersistentVolumeClaim{}
		if err = p.Cache.Create(LocalFFDBPVC, nn, pvc); err != nil {
			return nil, err
		}

		provutils.MakeLocalDBPVC(pvc, nn, p.Env, sizing.GetDefaultVolCapacity())

		if err = p.Cache.Update(LocalFFDBPVC, pvc); err != nil {
			return nil, err
		}
	}

	return ffp, nil
}

// CreateDatabase ensures a database is created for the given app.  The
// namespaced name passed in must be the actual name of the db resources
func (ff *localFeatureFlagsProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	c.FeatureFlags = &ff.Config
	return nil
}

func makeLocalFeatureFlags(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
	nn := providers.GetNamespacedName(o, "featureflags")

	dd := objMap[LocalFFDeployment].(*apps.Deployment)
	svc := objMap[LocalFFService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	port := int32(4242)

	envVars := []core.EnvVar{{
		Name: "DATABASE_URL",
		ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{
				LocalObjectReference: core.LocalObjectReference{
					Name: "featureflags-db",
				},
				Key: "connectionURL",
			},
		},
	},
	}

	ports := []core.ContainerPort{{
		Name:          "service",
		ContainerPort: port,
		Protocol:      "TCP",
	}}

	probeHandler := core.ProbeHandler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 4242,
			},
		},
	}

	livenessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 10,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	c := core.Container{
		Name:                     nn.Name,
		Image:                    IMAGE_FEATUREFLAGS_UNLEASH,
		Env:                      envVars,
		Ports:                    ports,
		LivenessProbe:            &livenessProbe,
		ReadinessProbe:           &readinessProbe,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
		Resources: core.ResourceRequirements{
			Limits: core.ResourceList{
				"memory": resource.MustParse("200Mi"),
				"cpu":    resource.MustParse("100m"),
			},
			Requests: core.ResourceList{
				"memory": resource.MustParse("100Mi"),
				"cpu":    resource.MustParse("50m"),
			},
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:       "featureflags",
		Port:       port,
		Protocol:   "TCP",
		TargetPort: intstr.FromInt(int(port)),
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)
}
