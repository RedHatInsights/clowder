package providers

import (
	"context"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MinIO struct {
	Client *minio.Client
	Config *config.ObjectStoreConfig
}

func GetConfig(ctx context.Context, m crd.MinioStatus, c client.Client) (*config.ObjectStoreConfig, error) {
	conf := &config.ObjectStoreConfig{
		Hostname: m.Hostname,
		Port:     int(m.Port),
	}

	name := types.NamespacedName{
		Name:      m.Credentials.Name,
		Namespace: m.Credentials.Namespace,
	}

	secret := core.Secret{}

	found, err := utils.UpdateOrErr(c.Get(ctx, name, &secret))

	if err != nil {
		return conf, err
	}

	if !found {
		return nil, nil
	}

	conf.AccessKey = string(secret.Data["accessKey"])
	conf.SecretKey = string(secret.Data["secretKey"])

	return conf, nil
}

// NewMinIO constructs a new client for the given config
func (m *MinIO) Init(ctx ProviderContext) (ctrl.Result, error) {
	result := ctrl.Result{}
	cfg, err := GetConfig(ctx.Ctx, ctx.Env.Status.ObjectStore.Minio, ctx.Client)

	if err != nil {
		return result, err
	}

	if cfg == nil {
		m.DeployMinio(types.NamespacedName{
			Namespace: ctx.Env.Spec.Namespace,
			Name:      ctx.Env.Name,
		})
	}

	endpoint := fmt.Sprintf("%v:%v", cfg.Hostname, cfg.Port)
	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})

	if err != nil {
		return result, err
	}

	m.Client = cl
	m.Config = cfg

	return result, nil
}

func (m *MinIO) Configure(c *config.AppConfig) {
	c.ObjectStore = m.Config
}

// CreateBucket creates a new bucket
func (m *MinIO) CreateBucket(ctx context.Context, bucket string) error {
	found, err := m.Client.BucketExists(ctx, bucket)

	if err != nil {
		return err
	}

	if found {
		return nil // possibly return a found error?
	}

	return m.Client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

// DeployMinio creates the actual minio service to be used by clowdapps, this
// does not create buckets
func (m *MinIO) DeployMinio(name types.NamespacedName, ctx context.Context, client client.Client) (ctrl.Result, error) {
	result := ctrl.Result{}

	dd := apps.Deployment{}
	update, err := utils.UpdateOrErr(client.Get(ctx, name, dd))

	if err != nil {
		return result, err
	}

	labels := m.Env.GetLabels()
	labels["env-app"] = nn.Name

	labeler := m.MakeLabeler(nn, labels)

	labeler(&dd)

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

	secret := &core.Secret{}
	secretUpdate, err := m.Get(nn, secret)
	if err != nil {
		return result, err
	}

	if len(secret.Data) == 0 {
		hostname := fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace)
		port := int32(9000)
		secret.StringData = map[string]string{
			"accessKey": utils.RandString(12),
			"secretKey": utils.RandString(12),
			"hostname":  hostname,
			"port":      strconv.Itoa(int(port)),
		}

		secret.Name = nn.Name
		secret.Namespace = nn.Namespace
		secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{m.Env.MakeOwnerReference()}
		secret.Type = core.SecretTypeOpaque

		if result, err = secretUpdate.Apply(secret); err != nil {
			return result, err
		}

		m.Env.Status.ObjectStore = crd.ObjectStoreStatus{
			Buckets: []string{},
			Minio: crd.MinioStatus{
				Credentials: core.SecretReference{
					Name:      secret.Name,
					Namespace: secret.Namespace,
				},
				Hostname: hostname,
				Port:     port,
			},
		}

		err = m.Client.Status().Update(m.Ctx, m.Env)

		if err != nil {
			return result, err
		}
	}

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
		ContainerPort: 9000,
	}}

	// TODO Readiness and Liveness probes

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
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	if result, err = update.Apply(&dd); err != nil {
		return result, err
	}

	s := core.Service{}
	update, err = m.Get(nn, &s)
	if err != nil {
		return result, err
	}

	servicePorts := []core.ServicePort{{
		Name:     "minio",
		Port:     9000,
		Protocol: "TCP",
	}}

	labeler(&s)

	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts

	if result, err = update.Apply(&s); err != nil {
		return result, err
	}

	pvc := core.PersistentVolumeClaim{}

	update, err = m.Get(nn, &pvc)
	if err != nil {
		return result, err
	}

	labeler(&pvc)

	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if result, err = update.Apply(&pvc); err != nil {
		return result, err
	}
	return result, nil
}
