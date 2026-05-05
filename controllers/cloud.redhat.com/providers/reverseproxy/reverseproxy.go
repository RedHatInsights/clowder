package reverseproxy

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// ReverseProxyDeployment is the resource ident for the reverse proxy deployment object.
var ReverseProxyDeployment = rc.NewSingleResourceIdent(ProvName, "reverse_proxy_deployment", &apps.Deployment{})

// ReverseProxyService is the resource ident for the reverse proxy service object.
var ReverseProxyService = rc.NewSingleResourceIdent(ProvName, "reverse_proxy_service", &core.Service{})

type localReverseProxyProvider struct {
	providers.Provider
}

// NewLocalReverseProxy constructs a new reverse proxy provider for ephemeral environments.
func NewLocalReverseProxy(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		ReverseProxyDeployment,
		ReverseProxyService,
	)

	cacheMap := []rc.ResourceIdent{
		ReverseProxyDeployment,
		ReverseProxyService,
	}

	err := providers.CachedMakeComponent(p, cacheMap, p.Env, "reverse-proxy", makeLocalReverseProxy, false)
	if err != nil {
		raisedErr := errors.Wrap("Couldn't make reverse proxy component", err)
		raisedErr.Requeue = true
		return nil, raisedErr
	}

	return &localReverseProxyProvider{Provider: *p}, nil
}

func (rp *localReverseProxyProvider) EnvProvide() error {
	return nil
}

func (rp *localReverseProxyProvider) Provide(_ *crd.ClowdApp) error {
	return nil
}

func makeLocalReverseProxy(_ *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, _ bool, nodePort bool) error {
	nn := providers.GetNamespacedName(o, "reverse-proxy")
	minioNN := providers.GetNamespacedName(o, "minio")

	dd := objMap[ReverseProxyDeployment].(*apps.Deployment)
	svc := objMap[ReverseProxyService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)
	labeler(dd)

	replicas := int32(1)
	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Labels = labels

	env, ok := o.(*crd.ClowdEnvironment)
	if !ok {
		return fmt.Errorf("could not get env from object")
	}

	bucketPathPrefix := "frontend-pushcache"
	if env.Spec.Providers.ReverseProxy.BucketPathPrefix != "" {
		bucketPathPrefix = env.Spec.Providers.ReverseProxy.BucketPathPrefix
	}

	spaEntrypointPath := "/data/chrome/index.html"
	if env.Spec.Providers.ReverseProxy.SpaEntrypointPath != "" {
		spaEntrypointPath = env.Spec.Providers.ReverseProxy.SpaEntrypointPath
	}

	awsRegion := "us-east-1"
	if env.Spec.Providers.ReverseProxy.AwsRegion != "" {
		awsRegion = env.Spec.Providers.ReverseProxy.AwsRegion
	}

	minioUpstreamURL := fmt.Sprintf("http://%s.%s.svc:9000", minioNN.Name, minioNN.Namespace)

	port := int32(8080)

	envVars := []core.EnvVar{
		{Name: "SERVER_PORT", Value: "8080"},
		{Name: "MINIO_UPSTREAM_URL", Value: minioUpstreamURL},
		{Name: "BUCKET_PATH_PREFIX", Value: bucketPathPrefix},
		{Name: "SPA_ENTRYPOINT_PATH", Value: spaEntrypointPath},
		{Name: "AWS_REGION", Value: awsRegion},
		{Name: "LOG_LEVEL", Value: "debug"},
	}

	envVars = provutils.AppendEnvVarsFromSecret(envVars, minioNN.Name,
		provutils.NewSecretEnvVar("PUSHCACHE_AWS_ACCESS_KEY_ID", "accessKey"),
		provutils.NewSecretEnvVar("PUSHCACHE_AWS_SECRET_ACCESS_KEY", "secretKey"),
	)

	ports := []core.ContainerPort{{
		Name:          "proxy",
		ContainerPort: port,
		Protocol:      core.ProtocolTCP,
	}}

	probeHandler := core.ProbeHandler{
		HTTPGet: &core.HTTPGetAction{
			Path: "/healthz",
			Port: intstr.FromInt(int(port)),
		},
	}

	livenessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 10,
		TimeoutSeconds:      2,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	readinessProbe := core.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 5,
		TimeoutSeconds:      2,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	c := core.Container{
		Name:                     nn.Name,
		Image:                    GetReverseProxyImage(env),
		Env:                      envVars,
		Ports:                    ports,
		LivenessProbe:            &livenessProbe,
		ReadinessProbe:           &readinessProbe,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:       "proxy",
		Port:       port,
		Protocol:   "TCP",
		TargetPort: intstr.FromInt(int(port)),
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)

	return nil
}
