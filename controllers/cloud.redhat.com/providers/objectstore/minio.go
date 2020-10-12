package objectstore

import (
	"fmt"
	"strconv"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const bucketCheckErrorMsg = "Failed to check if bucket %q exists in minio"
const bucketCreateErrorMsg = "Failed to create bucket %q in minio"

func wrapBucketError(errorMsg string, bucketName string, err error) error {
	msg := fmt.Sprintf(errorMsg, bucketName)
	return errors.Wrap(msg, err)
}

// minio is an object store provider that deploys and configures MinIO
type minioProvider struct {
	p.Provider
	Config config.ObjectStoreConfig
	Client *minio.Client
}

func (m *minioProvider) Configure(c *config.AppConfig) {
	c.ObjectStore = &config.ObjectStoreConfig{
		Hostname:  m.Config.Hostname,
		Port:      m.Config.Port,
		AccessKey: m.Config.AccessKey,
		SecretKey: m.Config.SecretKey,
		Buckets:   m.Config.Buckets,
	}
}

// CreateBuckets creates new buckets
func (m *minioProvider) CreateBuckets(app *crd.ClowdApp) error {
	for _, bucket := range app.Spec.ObjectStore {
		found, err := m.Client.BucketExists(m.Ctx, bucket)

		if err != nil {
			return wrapBucketError(bucketCheckErrorMsg, bucket, err)
		}

		if found {
			continue // possibly return a found error?
		}

		err = m.Client.MakeBucket(m.Ctx, bucket, minio.MakeBucketOptions{})

		if err != nil {
			return wrapBucketError(bucketCreateErrorMsg, bucket, err)
		}

		m.Config.Buckets = append(m.Config.Buckets, config.ObjectStoreBucket{
			Name:          bucket,
			RequestedName: bucket,
		})
	}

	return nil
}

// NewMinIO constructs a new minio for the given config
func NewMinIO(p *p.Provider) (ObjectStoreProvider, error) {
	cfg := config.ObjectStoreConfig{}

	minioProvider := minioProvider{Provider: *p, Config: cfg}

	nn := providers.GetNamespacedName(p.Env, "minio")

	dataInit := func() map[string]string {
		return map[string]string{
			"accessKey": utils.RandString(12),
			"secretKey": utils.RandString(12),
			"hostname":  fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
			"port":      strconv.Itoa(int(9000)),
		}
	}

	secMap, err := config.MakeOrGetSecret(p.Ctx, p.Env, p.Client, nn, dataInit) //Will set data if it already exists
	if err != nil {
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}

	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi((*secMap)["port"])
	minioProvider.Ctx = p.Ctx
	minioProvider.Config.AccessKey = providers.StrPtr((*secMap)["accessKey"])
	minioProvider.Config.SecretKey = providers.StrPtr((*secMap)["secretKey"])
	minioProvider.Config.Hostname = (*secMap)["hostname"]
	minioProvider.Config.Port = port
	endpoint := fmt.Sprintf("%v:%v", minioProvider.Config.Hostname, minioProvider.Config.Port)

	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(*minioProvider.Config.AccessKey, *minioProvider.Config.SecretKey, ""),
		Secure: false,
	})

	if err != nil {
		return nil, errors.Wrap("Failed to create minio client", err)
	}

	minioProvider.Client = cl

	providers.MakeComponent(p.Ctx, p.Client, p.Env, "minio", makeLocalMinIO)

	return &minioProvider, nil
}

func makeLocalMinIO(o utils.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim) {
	nn := providers.GetNamespacedName(o, "minio")

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: nn.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: nn.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = labels

	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{{
		Name: "quay-cloudservices-pull",
	}}

	// get the secret

	port := int32(9000)

	envVars := []core.EnvVar{{
		Name: "MINIO_ACCESS_KEY",
		ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{
				LocalObjectReference: core.LocalObjectReference{
					Name: nn.Name,
				},
				Key: "accessKey",
			},
		},
	}, {
		Name: "MINIO_SECRET_KEY",
		ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{
				LocalObjectReference: core.LocalObjectReference{
					Name: nn.Name,
				},
				Key: "secretKey",
			},
		},
	}}

	ports := []core.ContainerPort{{
		Name:          "minio",
		ContainerPort: port,
	}}

	probeHandler := core.Handler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 9000,
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

	c := core.Container{
		Name:  nn.Name,
		Image: "minio/minio",
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
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:     "minio",
		Port:     port,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o)
	utils.MakePVC(pvc, nn, labels, "1Gi", o)
}
