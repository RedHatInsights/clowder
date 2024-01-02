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
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/web"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// LocalFFDeployment is the ident referring to the local Feature Flags deployment object.
var LocalFFDeployment = rc.NewSingleResourceIdent(ProvName, "ff_deployment", &apps.Deployment{})

// LocalFFService is the ident referring to the local Feature Flags service object.
var LocalFFService = rc.NewSingleResourceIdent(ProvName, "ff_service", &core.Service{})

// LocalFFSecret is the ident referring to the local Feature Flags secret object.
var LocalFFSecret = rc.NewSingleResourceIdent(ProvName, "ff_secret", &core.Secret{})

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
}

// NewLocalFeatureFlagsProvider returns a new local featureflags provider object.
func NewLocalFeatureFlagsProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		LocalFFDeployment,
		LocalFFService,
		LocalFFSecret,
		LocalFFDBDeployment,
		LocalFFDBService,
		LocalFFDBPVC,
		LocalFFDBSecret,
	)
	return &localFeatureFlagsProvider{Provider: *p}, nil
}

func (ff *localFeatureFlagsProvider) EnvProvide() error {

	dataInit := createDefaultFFSecMap

	namespacedName := providers.GetNamespacedName(ff.Env, "featureflags")

	_, err := providers.MakeOrGetSecret(ff.Env, ff.Cache, LocalFFSecret, namespacedName, dataInit)
	if err != nil {
		raisedErr := errors.Wrap("Couldn't set/get secret", err)
		raisedErr.Requeue = true
		return raisedErr
	}

	objList := []rc.ResourceIdent{
		LocalFFDeployment,
		LocalFFService,
	}

	if err := providers.CachedMakeComponent(ff.Cache, objList, ff.Env, "featureflags", makeLocalFeatureFlags, false, ff.Env.IsNodePort()); err != nil {
		return err
	}

	namespacedNameDb := types.NamespacedName{
		Name:      "featureflags-db",
		Namespace: ff.Env.Status.TargetNamespace,
	}

	dd := &apps.Deployment{}
	if err := ff.Cache.Create(LocalFFDBDeployment, namespacedNameDb, dd); err != nil {
		return err
	}

	dbCfg := config.DatabaseConfig{}

	password, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return errors.Wrap("password generate failed", err)
	}

	pgPassword, err := utils.RandPassword(16, provutils.RCharSet)
	if err != nil {
		return errors.Wrap("pgPassword generate failed", err)
	}

	username := utils.RandString(16)
	hostname := fmt.Sprintf("%v.%v.svc", namespacedNameDb.Name, namespacedNameDb.Namespace)

	dataInitDb := func() map[string]string {

		return map[string]string{
			"hostname": hostname,
			"port":     "5432",
			"username": username,
			"password": password,
			"pgPass":   pgPassword,
			"name":     "unleash",
		}
	}

	secMapDb, err := providers.MakeOrGetSecret(ff.Env, ff.Cache, LocalFFDBSecret, namespacedNameDb, dataInitDb)
	if err != nil {
		return errors.Wrap("Couldn't set/get secret", err)
	}

	err = dbCfg.Populate(secMapDb)
	if err != nil {
		return errors.Wrap("couldn't convert to int", err)
	}
	dbCfg.AdminUsername = "postgres"

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

	provutils.MakeLocalDB(dd, namespacedNameDb, ff.Env, labels, &dbCfg, "quay.io/cloudservices/postgresql-rds:15-53ac80c", ff.Env.Spec.Providers.FeatureFlags.PVC, "unleash", &res)

	if err = ff.Cache.Update(LocalFFDBDeployment, dd); err != nil {
		return err
	}

	s := &core.Service{}
	if err := ff.Cache.Create(LocalFFDBService, namespacedNameDb, s); err != nil {
		return err
	}

	provutils.MakeLocalDBService(s, namespacedNameDb, ff.Env, labels)

	if err = ff.Cache.Update(LocalFFDBService, s); err != nil {
		return err
	}

	if ff.Env.Spec.Providers.FeatureFlags.PVC {
		pvc := &core.PersistentVolumeClaim{}
		if err = ff.Cache.Create(LocalFFDBPVC, namespacedNameDb, pvc); err != nil {
			return err
		}

		provutils.MakeLocalDBPVC(pvc, namespacedNameDb, ff.Env, sizing.GetDefaultVolCapacity())

		if err = ff.Cache.Update(LocalFFDBPVC, pvc); err != nil {
			return err
		}
	}

	return nil
}

func createDefaultFFSecMap() map[string]string {

	randString := utils.RandHexString(32)

	return map[string]string{
		"adminAccessToken":  "*:*." + randString,
		"clientAccessToken": "*:default." + randString + ",*:development." + randString + ",*:production." + randString,
	}
}

// CreateDatabase ensures a database is created for the given app.  The
// namespaced name passed in must be the actual name of the db resources
func (ff *localFeatureFlagsProvider) Provide(_ *crd.ClowdApp) error {

	secret := &core.Secret{}
	nn := providers.GetNamespacedName(ff.Env, "featureflags")

	if err := ff.Client.Get(ff.Ctx, nn, secret); err != nil {
		return err
	}

	ff.Config.FeatureFlags = &config.FeatureFlagsConfig{
		Hostname:          fmt.Sprintf("%s-featureflags.%s.svc", ff.Env.Name, ff.Env.Status.TargetNamespace),
		Port:              4242,
		Scheme:            config.FeatureFlagsConfigSchemeHttp,
		ClientAccessToken: utils.StringPtr(string(secret.Data["clientAccessToken"])),
	}

	return nil
}

func makeLocalFeatureFlags(cache *rc.ObjectCache, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) {
	nn := providers.GetNamespacedName(o, "featureflags")
	dd := objMap[LocalFFDeployment].(*apps.Deployment)
	svc := objMap[LocalFFService].(*core.Service)
	environment := o.(*crd.ClowdEnvironment)
	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labels["service"] = "featureflags"
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	port := int32(4242)

	envVars := []core.EnvVar{
		{
			Name:  "DATABASE_SSL",
			Value: "false",
		},
		{
			Name:  "KC_HOST",
			Value: web.GetAuthHostname(environment.Status.Hostname),
		},
		{
			Name:  "KC_REALM",
			Value: "unleash",
		},
		{
			Name:  "KC_CLIENT_ID",
			Value: "unleash",
		},
		{
			Name:  "KC_ADMIN_ROLES",
			Value: "admin",
		},
		{
			Name:  "KC_EDITOR_ROLES",
			Value: "editor",
		},
		{
			Name:  "KC_VIEWER_ROLES",
			Value: "viewer",
		},
		{
			Name:  "KC_CLIENT_SECRET",
			Value: "notsosecret",
		},
	}

	envVars = provutils.AppendEnvVarsFromSecret(envVars, "featureflags-db",
		provutils.NewSecretEnvVar("DATABASE_HOST", "hostname"),
		provutils.NewSecretEnvVar("DATABASE_PORT", "port"),
		provutils.NewSecretEnvVar("DATABASE_USERNAME", "username"),
		provutils.NewSecretEnvVar("DATABASE_PASSWORD", "password"),
		provutils.NewSecretEnvVar("DATABASE_NAME", "name"),
	)
	envVars = provutils.AppendEnvVarsFromSecret(envVars, nn.Name,
		provutils.NewSecretEnvVar("INIT_CLIENT_API_TOKENS", "clientAccessToken"),
		provutils.NewSecretEnvVar("INIT_ADMIN_API_TOKENS", "adminAccessToken"),
	)

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
		Image:                    DefaultImageFeatureFlagsUnleash,
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
