package providers

import (
	"context"
	"fmt"
	"strconv"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

// MinIO is an object store provider that deploys and configures MinIO
type MinIO struct {
	Client *minio.Client
	Config *config.ObjectStoreConfig
}

// Init constructs a new client for the given config
func (m *MinIO) Init(ctx ProviderContext) error {
	cfg, err := GetConfig(ctx.Ctx, ctx.Env.Status.ObjectStore.Minio, ctx.Client)

	if err != nil {
		return err
	}

	if cfg == nil {
		m.DeployMinio(
			ctx.Ctx,
			types.NamespacedName{
				Namespace: ctx.Env.Spec.Namespace,
				Name:      ctx.Env.Name,
			},
			ctx.Client,
			ctx.Env,
		)
	}

	endpoint := fmt.Sprintf("%v:%v", cfg.Hostname, cfg.Port)
	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})

	if err != nil {
		return err
	}

	m.Client = cl
	m.Config = cfg

	return nil
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
func (m *MinIO) DeployMinio(ctx context.Context, nn types.NamespacedName, client client.Client, env *crd.ClowdEnvironment) (ctrl.Result, error) {
	result := ctrl.Result{}

	dd := apps.Deployment{}
	update, err := utils.UpdateOrErr(client.Get(ctx, nn, &dd))

	if err != nil {
		return result, err
	}

	labels := env.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, env)

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
	secretUpdate, err := utils.UpdateOrErr(client.Get(ctx, nn, secret))

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
		secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{env.MakeOwnerReference()}
		secret.Type = core.SecretTypeOpaque

		if result, err = secretUpdate.Apply(ctx, client, secret); err != nil {
			return result, err
		}

		env.Status.ObjectStore = crd.ObjectStoreStatus{
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

		err = client.Status().Update(ctx, env)

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

	if result, err = update.Apply(ctx, client, &dd); err != nil {
		return result, err
	}

	s := core.Service{}
	update, err = utils.UpdateOrErr(client.Get(ctx, nn, &s))

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

	if result, err = update.Apply(ctx, client, &s); err != nil {
		return result, err
	}

	pvc := core.PersistentVolumeClaim{}

	update, err = utils.UpdateOrErr(client.Get(ctx, nn, &pvc))

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

	if result, err = update.Apply(ctx, client, &pvc); err != nil {
		return result, err
	}
	return result, nil
}
