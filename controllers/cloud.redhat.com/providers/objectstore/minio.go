package objectstore

import (
	"context"
	"fmt"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// MinioDeployment is the resource ident for the Minio deployment object.
var MinioDeployment = rc.NewSingleResourceIdent(ProvName, "minio_db_deployment", &apps.Deployment{})

// MinioService is the resource ident for the Minio service object.
var MinioService = rc.NewSingleResourceIdent(ProvName, "minio_db_service", &core.Service{})

// MinioPVC is the resource ident for the Minio PVC object.
var MinioPVC = rc.NewSingleResourceIdent(ProvName, "minio_db_pvc", &core.PersistentVolumeClaim{})

// MinioSecret is the resource ident for the Minio secret object.
var MinioSecret = rc.NewSingleResourceIdent(ProvName, "minio_db_secret", &core.Secret{})

// MinioNetworkPolicy is the resource ident for the KafkaNetworkPolicy
var MinioNetworkPolicy = rc.NewSingleResourceIdent(ProvName, "minio_network_policy", &networking.NetworkPolicy{})

// minio is an object store provider that deploys and configures MinIO
type minioProvider struct {
	providers.Provider
	BucketHandler bucketHandler
}

// NewMinIO constructs a new minio for the given config
func NewMinIO(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		MinioDeployment,
		MinioService,
		MinioPVC,
		MinioSecret,
		MinioNetworkPolicy,
	)

	nn := providers.GetNamespacedName(p.Env, "minio")

	dataInit := func() map[string]string {
		return createDefaultMinioSecMap(nn.Name, nn.Namespace)
	}
	// MakeOrGetSecret will set data if it already exists
	secMap, err := providers.MakeOrGetSecret(p.Env, p.Cache, MinioSecret, nn, dataInit)
	if err != nil {
		raisedErr := errors.Wrap("Couldn't set/get secret", err)
		raisedErr.Requeue = true
		return nil, raisedErr
	}

	mp, err := createMinioProvider(p, *secMap, &minioHandler{})

	if err != nil {
		return nil, err
	}

	minioCacheMap := []rc.ResourceIdent{
		MinioDeployment,
		MinioService,
	}

	if p.Env.Spec.Providers.ObjectStore.PVC {
		minioCacheMap = append(minioCacheMap, MinioPVC)
	}

	err = providers.CachedMakeComponent(p, minioCacheMap, p.Env, "minio", makeLocalMinIO, p.Env.Spec.Providers.ObjectStore.PVC)

	if err != nil {
		raisedErr := errors.Wrap("Couldn't make component", err)
		raisedErr.Requeue = true
		return nil, raisedErr
	}

	return mp, nil
}

func (m *minioProvider) EnvProvide() error {
	return createNetworkPolicy(&m.Provider)
}

// Provide creates new buckets
func (m *minioProvider) Provide(app *crd.ClowdApp) error {
	if len(app.Spec.ObjectStore) == 0 {
		return nil
	}

	secret := &core.Secret{}
	nn := providers.GetNamespacedName(m.Env, "minio")

	if err := m.Client.Get(m.Ctx, nn, secret); err != nil {
		return err
	}

	if _, err := m.HashCache.CreateOrUpdateObject(secret, true); err != nil {
		return err
	}

	if err := m.HashCache.AddClowdObjectToObject(m.Env, secret); err != nil {
		return err
	}

	port, err := strconv.Atoi(string(secret.Data["port"]))
	if err != nil {
		return err
	}

	m.Config.ObjectStore = &config.ObjectStoreConfig{
		Hostname:  string(secret.Data["hostname"]),
		Port:      int(port),
		AccessKey: utils.StringPtr(string(secret.Data["accessKey"])),
		SecretKey: utils.StringPtr(string(secret.Data["secretKey"])),
		Tls:       false,
		Buckets:   []config.ObjectStoreBucket{},
	}

	for _, bucket := range app.Spec.ObjectStore {
		found, err := m.BucketHandler.Exists(m.Ctx, bucket)

		if err != nil {
			return newBucketError(bucketCheckErrorMsg, bucket, err)
		}

		if !found {
			err = m.BucketHandler.Make(m.Ctx, bucket)

			if err != nil {
				return newBucketError(bucketCreateErrorMsg, bucket, err)
			}
		}

		newBucket := config.ObjectStoreBucket{
			Name:          bucket,
			RequestedName: bucket,
			Endpoint:      utils.StringPtr(string(secret.Data["hostname"])),
		}

		if string(secret.Data["accessKey"]) != "" {
			newBucket.AccessKey = m.Config.ObjectStore.AccessKey
		}
		if string(secret.Data["secretKey"]) != "" {
			newBucket.SecretKey = m.Config.ObjectStore.SecretKey
		}

		m.Config.ObjectStore.Buckets = append(m.Config.ObjectStore.Buckets, newBucket)
	}

	return nil
}

const bucketCheckErrorMsg = "failed to check if bucket exists"
const bucketCreateErrorMsg = "failed to create bucket"

func newBucketError(msg string, bucketName string, rootCause error) error {
	newErr := errors.Wrap(fmt.Sprintf("bucket %q -- %s", bucketName, msg), rootCause)
	newErr.Requeue = true
	return newErr
}

// Create a bucketHandler interface to allow for mocking of minio client actions in tests
type bucketHandler interface {
	Exists(ctx context.Context, bucketName string) (bool, error)
	Make(ctx context.Context, bucketName string) error
	CreateClient(hostname string, port int, accessKey *string, secretKey *string) error
}

// minioHandler will implement the above interface using minio-go
type minioHandler struct {
	Client *minio.Client
}

func (h *minioHandler) Exists(ctx context.Context, bucketName string) (bool, error) {
	return h.Client.BucketExists(ctx, bucketName)
}

func (h *minioHandler) Make(ctx context.Context, bucketName string) error {
	return h.Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
}

func (h *minioHandler) CreateClient(
	hostname string, port int, accessKey *string, secretKey *string,
) error {
	endpoint := fmt.Sprintf("%v:%v", hostname, port)

	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(*accessKey, *secretKey, ""),
		Secure: false,
	})

	if err != nil {
		return errors.Wrap("Failed to create minio client", err)
	}

	h.Client = cl

	return nil
}

func createMinioProvider(
	p *providers.Provider, secMap map[string]string, handler bucketHandler,
) (*minioProvider, error) {
	mp := &minioProvider{Provider: *p}

	port, err := strconv.Atoi(secMap["port"])
	if err != nil {
		return nil, err
	}
	mp.Ctx = p.Ctx

	mp.BucketHandler = handler
	err = mp.BucketHandler.CreateClient(
		secMap["hostname"],
		int(port),
		utils.StringPtr(secMap["accessKey"]),
		utils.StringPtr(secMap["secretKey"]),
	)

	if err != nil {
		return nil, errors.Wrap("error creating minio client", err)
	}
	return mp, nil
}

func createDefaultMinioSecMap(name string, namespace string) map[string]string {
	return map[string]string{
		"accessKey": utils.RandString(12),
		"secretKey": utils.RandString(12),
		"hostname":  fmt.Sprintf("%v.%v.svc", name, namespace),
		"port":      strconv.Itoa(int(9000)),
	}
}

func createNetworkPolicy(p *providers.Provider) error {
	clowderNs, err := provutils.GetClowderNamespace()

	if err != nil {
		return nil
	}

	np := &networking.NetworkPolicy{}
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("allow-from-%s-namespace", clowderNs),
		Namespace: p.Env.Status.TargetNamespace,
	}

	if err := p.Cache.Create(MinioNetworkPolicy, nn, np); err != nil {
		return err
	}

	npFrom := []networking.NetworkPolicyPeer{{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": clowderNs,
			},
		},
	}}

	np.Spec.Ingress = []networking.NetworkPolicyIngressRule{{
		From: npFrom,
	}}

	np.Spec.PolicyTypes = []networking.PolicyType{"Ingress"}

	labeler := utils.GetCustomLabeler(nil, nn, p.Env)
	labeler(np)

	return p.Cache.Update(MinioNetworkPolicy, np)
}

func makeLocalMinIO(_ *crd.ClowdEnvironment, o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) error {
	nn := providers.GetNamespacedName(o, "minio")

	dd := objMap[MinioDeployment].(*apps.Deployment)
	svc := objMap[MinioService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	var volSource core.VolumeSource
	if usePVC {
		volSource = core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: nn.Name,
			},
		}
	} else {
		volSource = core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		}
	}

	dd.Spec.Template.Spec.Volumes = []core.Volume{
		{
			Name:         nn.Name,
			VolumeSource: volSource,
		},
	}
	dd.Spec.Template.ObjectMeta.Labels = labels

	port := int32(9000)

	envVars := []core.EnvVar{}

	envVars = provutils.AppendEnvVarsFromSecret(envVars, nn.Name,
		provutils.NewSecretEnvVar("MINIO_ACCESS_KEY", "accessKey"),
		provutils.NewSecretEnvVar("MINIO_SECRET_KEY", "secretKey"),
	)

	ports := []core.ContainerPort{{
		Name:          "minio",
		ContainerPort: port,
		Protocol:      core.ProtocolTCP,
	}}

	probeHandler := core.ProbeHandler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 9000,
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

	env, ok := o.(*crd.ClowdEnvironment)
	if !ok {
		return fmt.Errorf("could not get env from object")
	}

	c := core.Container{
		Name:  nn.Name,
		Image: GetObjectStoreMinioImage(env),
		Env:   envVars,
		Ports: ports,
		VolumeMounts: []core.VolumeMount{{
			Name:      nn.Name,
			MountPath: "/storage",
		}},
		Args: []string{
			"server",
			"/storage",
		},
		LivenessProbe:            &livenessProbe,
		ReadinessProbe:           &readinessProbe,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
		ImagePullPolicy:          core.PullIfNotPresent,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:       "minio",
		Port:       port,
		Protocol:   "TCP",
		TargetPort: intstr.FromInt(int(port)),
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)
	if usePVC {
		pvc := objMap[MinioPVC].(*core.PersistentVolumeClaim)
		utils.MakePVC(pvc, nn, labels, "1Gi", o)
	}
	return nil
}
