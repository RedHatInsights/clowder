package featureflags

import (
	"fmt"
	"net/url"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

const featureFlagsPort = 4242
const featureFlagsEdgePort = 3063

// LocalFFDeployment is the ident referring to the local Feature Flags deployment object.
var LocalFFDeployment = rc.NewSingleResourceIdent(ProvName, "ff_deployment", &apps.Deployment{})

// LocalFFService is the ident referring to the local Feature Flags service object.
var LocalFFService = rc.NewSingleResourceIdent(ProvName, "ff_service", &core.Service{})

// LocalFFEdgeDeployment is the ident referring to the local Unleash edge deployment object.
var LocalFFEdgeDeployment = rc.NewSingleResourceIdent(ProvName, "ff_edge_deployment", &apps.Deployment{})

// LocalFFEdgeService is the ident referring to the local Unleash edge service object.
var LocalFFEdgeService = rc.NewSingleResourceIdent(ProvName, "ff_edge_service", &core.Service{})

// LocalFFEdgeIngress is the ident referring to the local Unleash edge ingress object.
var LocalFFEdgeIngress = rc.NewSingleResourceIdent(ProvName, "ff_edge_ingress", &networking.Ingress{})

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
		LocalFFEdgeDeployment,
		LocalFFEdgeService,
		LocalFFEdgeIngress,
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

	if err := providers.CachedMakeComponent(ff, objList, ff.Env, "featureflags", makeLocalFeatureFlags, false); err != nil {
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
	passwordEncode := url.QueryEscape(password)
	connectionURL := fmt.Sprintf("postgres://%s:%s@%s/%s", username, passwordEncode, hostname, "unleash")

	dataInitDb := func() map[string]string {

		return map[string]string{
			"hostname":      hostname,
			"port":          "5432",
			"username":      username,
			"password":      password,
			"pgPass":        pgPassword,
			"name":          "unleash",
			"connectionURL": connectionURL,
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

	dbImage, err := provutils.GetDefaultDatabaseImage(16, false)
	if err != nil {
		return err
	}

	provutils.MakeLocalDB(dd, namespacedNameDb, ff.Env, labels, &dbCfg, dbImage, ff.Env.Spec.Providers.FeatureFlags.PVC, "unleash", &res)

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

	objList2 := []rc.ResourceIdent{
		LocalFFEdgeDeployment,
		LocalFFEdgeService,
	}

	if err := providers.CachedMakeComponent(ff, objList2, ff.Env, "featureflags-edge", makeLocalFeatureFlagsEdge, false); err != nil {
		return err
	}

	if err := makeLocalFFEdgeIngress(ff); err != nil {
		return err
	}

	return nil
}

func createDefaultFFSecMap() map[string]string {
	return map[string]string{
		"adminAccessToken":  "*:*." + utils.RandHexString(32),
		"clientAccessToken": "default:development." + utils.RandHexString(32),
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

func makeLocalFFEdgeIngress(ff *localFeatureFlagsProvider) error {
	nn := providers.GetNamespacedName(ff.Env, "featureflags")
	nnEdge := providers.GetNamespacedName(ff.Env, "featureflags-edge")

	ingress := &networking.Ingress{}
	if err := ff.Cache.Create(LocalFFEdgeIngress, nn, ingress); err != nil {
		return err
	}

	labels := ff.Env.GetLabels()
	labeler := utils.MakeLabeler(nn, labels, ff.Env)
	labeler(ingress)

	ingressClass := ff.Env.Spec.Providers.Web.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	prefixPathType := networking.PathTypePrefix
	ingress.Spec = networking.IngressSpec{
		IngressClassName: &ingressClass,
		Rules: []networking.IngressRule{{
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{{
						Path:     "/api/client/features",
						PathType: &prefixPathType,
						Backend: networking.IngressBackend{
							Service: &networking.IngressServiceBackend{
								Name: nnEdge.Name,
								Port: networking.ServiceBackendPort{
									Name: "unleash-edge",
								},
							},
						},
					}},
				},
			},
		}},
	}

	if err := ff.Cache.Update(LocalFFEdgeIngress, ingress); err != nil {
		return err
	}
	return nil
}

func makeLocalFeatureFlags(_ *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) error {
	nn := providers.GetNamespacedName(o, "featureflags")

	dd := objMap[LocalFFDeployment].(*apps.Deployment)
	svc := objMap[LocalFFService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labels["service"] = "featureflags"
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.Labels = labels

	port := int32(featureFlagsPort)

	envVars := []core.EnvVar{
		{
			Name:  "DATABASE_SSL",
			Value: "false",
		},
	}

	envVars = provutils.AppendEnvVarsFromSecret(envVars, "featureflags-db",
		provutils.NewSecretEnvVar("DATABASE_URL", "connectionURL"),
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
		HTTPGet: &core.HTTPGetAction{
			Path: "/health",
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: featureFlagsPort,
			},
		},
	}

	livenessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      4,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 40,
		TimeoutSeconds:      4,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	env, ok := o.(*crd.ClowdEnvironment)
	if !ok {
		return fmt.Errorf("could not get env")
	}

	c := core.Container{
		Name:                     nn.Name,
		Image:                    GetFeatureFlagsUnleashImage(env),
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
	return nil
}

func makeLocalFeatureFlagsEdge(_ *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) error {

	nnFF := providers.GetNamespacedName(o, "featureflags")
	nn := providers.GetNamespacedName(o, "featureflags-edge")

	dd := objMap[LocalFFEdgeDeployment].(*apps.Deployment)
	svc := objMap[LocalFFEdgeService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nnFF.Name
	labels["service"] = "unleash-edge"
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.Labels = labels

	portEdge := int32(featureFlagsEdgePort)

	envVarsEdge := []core.EnvVar{
		{
			// communication with the main featureflags service on localhost
			Name:  "UPSTREAM_URL",
			Value: fmt.Sprintf("http://%s:%d", nnFF.Name, featureFlagsPort),
		},
	}
	envVarsEdge = provutils.AppendEnvVarsFromSecret(envVarsEdge, nnFF.Name,
		provutils.NewSecretEnvVar("TOKENS", "clientAccessToken"))

	portsEdge := []core.ContainerPort{{
		Name:          "service",
		ContainerPort: portEdge,
		Protocol:      "TCP",
	}}

	readinessProbeEdge := core.Probe{
		ProbeHandler: core.ProbeHandler{
			Exec: &core.ExecAction{
				Command: []string{"/unleash-edge", "ready"},
			},
		},
		InitialDelaySeconds: 1,
		TimeoutSeconds:      5,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	livenessProbeEdge := core.Probe{
		ProbeHandler: core.ProbeHandler{
			Exec: &core.ExecAction{
				Command: []string{"/unleash-edge", "health"},
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	env, ok := o.(*crd.ClowdEnvironment)
	if !ok {
		return fmt.Errorf("could not get env")
	}

	ce := core.Container{
		Name:            nn.Name,
		Image:           GetFeatureFlagsUnleashEdgeImage(env),
		Env:             envVarsEdge,
		Ports:           portsEdge,
		Args:            []string{"edge"},
		ReadinessProbe:  &readinessProbeEdge,
		LivenessProbe:   &livenessProbeEdge,
		ImagePullPolicy: core.PullIfNotPresent,
		Resources: core.ResourceRequirements{
			Requests: core.ResourceList{
				"memory": resource.MustParse("200Mi"),
				"cpu":    resource.MustParse("100m"),
			},
			Limits: core.ResourceList{
				"memory": resource.MustParse("400Mi"),
				"cpu":    resource.MustParse("200m"),
			},
		},
	}

	dd.Spec.Template.Spec.Containers = []core.Container{ce}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:       "unleash-edge",
		Port:       portEdge,
		Protocol:   "TCP",
		TargetPort: intstr.FromInt(int(portEdge)),
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)

	return nil
}
