package featureflags

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	provutils "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/utils"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type localFeatureFlagsProvider struct {
	p.Provider
	Config config.FeatureFlagsConfig
}

// NewLocalFeatureFlagsProvider returns a new local featureflags provider object.
func NewLocalFeatureFlagsProvider(p *p.Provider) (providers.ClowderProvider, error) {

	ffp := &localFeatureFlagsProvider{Provider: *p, Config: config.FeatureFlagsConfig{}}

	err := providers.MakeComponent(p.Ctx, p.Client, p.Env, "featureflags", makeLocalFeatureFlags, false)
	if err != nil {
		raisedErr := errors.Wrap("Couldn't make component", err)
		raisedErr.Requeue = true
		return nil, raisedErr
	}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("featureflags-db"),
		Namespace: p.Env.Status.TargetNamespace,
	}

	dd := apps.Deployment{}
	exists, err := utils.UpdateOrErr(p.Client.Get(p.Ctx, nn, &dd))

	if err != nil {
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

	secMap, err := config.MakeOrGetSecret(p.Ctx, p.Env, p.Client, nn, dataInit)
	if err != nil {
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}

	dbCfg.Populate(secMap)
	dbCfg.AdminUsername = "postgres"

	ffp.Config = config.FeatureFlagsConfig{
		Hostname: fmt.Sprintf("%s-featureflags.%s.svc", p.Env.Name, p.Env.Status.TargetNamespace),
		Port:     4242,
	}

	provutils.MakeLocalDB(&dd, nn, p.Env, &dbCfg, "registry.redhat.io/rhel8/postgresql-12:1-36", p.Env.Spec.Providers.FeatureFlags.PVC, "unleash")

	if err = exists.Apply(p.Ctx, p.Client, &dd); err != nil {
		return nil, err
	}

	s := core.Service{}
	update, err := utils.UpdateOrErr(p.Client.Get(p.Ctx, nn, &s))

	if err != nil {
		return nil, err
	}

	provutils.MakeLocalDBService(&s, nn, p.Env)

	if err = update.Apply(p.Ctx, p.Client, &s); err != nil {
		return nil, err
	}

	if p.Env.Spec.Providers.FeatureFlags.PVC {
		pvc := core.PersistentVolumeClaim{}
		update, err = utils.UpdateOrErr(p.Client.Get(p.Ctx, nn, &pvc))

		if err != nil {
			return nil, err
		}

		provutils.MakeLocalDBPVC(&pvc, nn, p.Env)

		if err = update.Apply(p.Ctx, p.Client, &pvc); err != nil {
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

func makeLocalFeatureFlags(o obj.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim, usePVC bool) {
	nn := providers.GetNamespacedName(o, "featureflags")

	labels := o.GetLabels()
	labels["env-clowdapp"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{{
		Name: "quay-cloudservices-pull",
	}}

	dd.Spec.Template.Spec.ServiceAccountName = o.GetClowdSAName()

	// get the secret

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
	}}

	probeHandler := core.Handler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 4242,
			},
		},
	}

	livenessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 10,
		TimeoutSeconds:      2,
	}
	readinessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      2,
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          "unleashorg/unleash-server:3.9",
		Env:            envVars,
		Ports:          ports,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:     "featureflags",
		Port:     port,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o)
}
