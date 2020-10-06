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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// minio is an object store provider that deploys and configures MinIO
type minioProvider struct {
	Ctx    context.Context
	Client *minio.Client
	Config *config.ObjectStoreConfig
}

func (m *minioProvider) Configure(c *config.AppConfig) {
	c.ObjectStore = m.Config
}

// CreateBuckets creates new buckets
func (m *minioProvider) CreateBuckets(app *crd.ClowdApp) error {
	for _, bucket := range app.Spec.ObjectStore {
		found, err := m.Client.BucketExists(m.Ctx, bucket)

		if err != nil {
			return errors.Wrap("Failed to check if bucket exists in minio", err)
		}

		if found {
			continue // possibly return a found error?
		}

		err = m.Client.MakeBucket(m.Ctx, bucket, minio.MakeBucketOptions{})

		if err != nil {
			msg := fmt.Sprintf("Failed to create bucket %s in minio", bucket)
			return errors.Wrap(msg, err)
		}
	}

	return nil
}

// NewMinIO constructs a new minio for the given config
func NewMinIO(p *Provider) (ObjectStoreProvider, error) {
	m := &minioProvider{Ctx: p.Ctx}
	nn := types.NamespacedName{
		Namespace: p.Env.Spec.Namespace,
		Name:      p.Env.Name + "-minio",
	}

	cfg, err := deployMinio(
		p.Ctx,
		nn,
		p.Client,
		p.Env,
	)

	if err != nil {
		return m, err
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

	// get the secret

	port := int32(9000)

	obsCfg := config.ObjectStoreConfig{}
	dataInit := func() map[string]string {
		return map[string]string{
			"accessKey": utils.RandString(12),
			"secretKey": utils.RandString(12),
			"hostname":  fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace),
			"port":      strconv.Itoa(int(port)),
		}
	}

	secMap, err := config.MakeOrGetSecret(ctx, env, client, nn, dataInit) //Will set data if it already exists
	if err != nil {
		return nil, errors.Wrap("Couldn't set/get secret", err)
	}
	obsCfg.Populate(secMap)

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

	utils.MakeService(&s, nn, labels, servicePorts, env)

	if err = update.Apply(ctx, client, &s); err != nil {
		return nil, err
	}

	pvc := core.PersistentVolumeClaim{}

	update, err = utils.UpdateOrErr(client.Get(ctx, nn, &pvc))

	if err != nil {
		return nil, err
	}

	utils.MakePVC(&pvc, nn, labels, "1Gi", env)

	if err = update.Apply(ctx, client, &pvc); err != nil {
		return nil, err
	}

	return &obsCfg, nil
}
