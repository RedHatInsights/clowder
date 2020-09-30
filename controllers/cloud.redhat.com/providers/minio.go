package providers

import (
	"context"
	"fmt"
	"strconv"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// minio is an object store provider that deploys and configures MinIO
type minioProvider struct {
	Ctx    context.Context
	Client *minio.Client
	Config *config.ObjectStoreConfig
}

func (m *minioProvider) Configure(c *config.AppConfig) {
	c.ObjectStore = m.Config
}

// CreateBucket creates a new bucket
func (m *minioProvider) CreateBucket(bucket string) error {
	found, err := m.Client.BucketExists(m.Ctx, bucket)

	if err != nil {
		return errors.Wrap("Failed to check if bucket exists in minio", err)
	}

	if found {
		return nil // possibly return a found error?
	}

	err = m.Client.MakeBucket(m.Ctx, bucket, minio.MakeBucketOptions{})

	if err != nil {
		return errors.Wrap("Failed to create bucket in minio", err)
	}

	return nil
}

// NewMinIO constructs a new minio for the given config
func NewMinIO(p *Provider) (ObjectStoreProvider, error) {
	m := &minioProvider{Ctx: p.Ctx}
	cfg, err := GetConfig(p.Ctx, p.Env.Status.ObjectStore.Minio, p.Client)

	if err != nil {
		return m, errors.Wrap("Failed to fetch minio config", err)
	}

	if cfg == nil {
		cfg, err = deployMinio(
			p.Ctx,
			types.NamespacedName{
				Namespace: p.Env.Spec.Namespace,
				Name:      p.Env.Name + "-minio",
			},
			p.Client,
			p.Env,
		)

		if err != nil {
			return m, err
		}
	}

	endpoint := fmt.Sprintf("%v:%v", cfg.Hostname, cfg.Port)
	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})

	if err != nil {
		return m, errors.Wrap("Failed to create minio client", err)
	}

	m.Client = cl
	m.Config = cfg

	return m, nil
}

// DeployMinio creates the actual minio service to be used by clowdapps, this
// does not create buckets
func deployMinio(ctx context.Context, nn types.NamespacedName, client client.Client, env *crd.ClowdEnvironment) (*config.ObjectStoreConfig, error) {
	dd := apps.Deployment{}
	update, err := utils.UpdateOrErr(client.Get(ctx, nn, &dd))

	if err != nil {
		return nil, err
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
		return nil, err
	}

	var hostname, accessKey, secretKey string
	port := int32(9000)

	if len(secret.Data) == 0 {
		hostname = fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace)
		accessKey = utils.RandString(12)
		secretKey = utils.RandString(12)
		secret.StringData = map[string]string{
			"accessKey": accessKey,
			"secretKey": secretKey,
			"hostname":  hostname,
			"port":      strconv.Itoa(int(port)),
		}

		secret.Name = nn.Name
		secret.Namespace = nn.Namespace
		secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{env.MakeOwnerReference()}
		secret.Type = core.SecretTypeOpaque

		if err = secretUpdate.Apply(ctx, client, secret); err != nil {
			return nil, err
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
			return nil, errors.Wrap("Failed to update minio status on env", err)
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

	if err = update.Apply(ctx, client, &dd); err != nil {
		return nil, err
	}

	s := core.Service{}
	update, err = utils.UpdateOrErr(client.Get(ctx, nn, &s))

	if err != nil {
		return nil, err
	}

	servicePorts := []core.ServicePort{{
		Name:     "minio",
		Port:     port,
		Protocol: "TCP",
	}}

	labeler(&s)

	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts

	if err = update.Apply(ctx, client, &s); err != nil {
		return nil, err
	}

	pvc := core.PersistentVolumeClaim{}

	update, err = utils.UpdateOrErr(client.Get(ctx, nn, &pvc))

	if err != nil {
		return nil, err
	}

	labeler(&pvc)

	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(ctx, client, &pvc); err != nil {
		return nil, err
	}

	return &config.ObjectStoreConfig{
		Hostname:  hostname,
		Port:      int(port),
		AccessKey: accessKey,
		SecretKey: secretKey,
	}, nil
}
