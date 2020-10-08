package objectstore

import (
	"context"
	"fmt"
	"strconv"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// minio is an object store provider that deploys and configures MinIO
type minioProvider struct {
	Ctx       context.Context
	Client    *minio.Client
	Buckets   []config.ObjectStoreBucket
	Hostname  string
	Port      int
	AccessKey string
	SecretKey string
}

func (m *minioProvider) Configure(c *config.AppConfig) {
	c.ObjectStore = &config.ObjectStoreConfig{
		Hostname:  m.Hostname,
		Port:      m.Port,
		AccessKey: &m.AccessKey,
		SecretKey: &m.SecretKey,
		Buckets:   m.Buckets,
	}
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

		m.Buckets = append(m.Buckets, config.ObjectStoreBucket{
			Name:          bucket,
			RequestedName: bucket,
		})
	}

	return nil
}

// NewMinIO constructs a new minio for the given config
func NewMinIO(p *p.Provider) (p.ObjectStoreProvider, error) {
	nn := types.NamespacedName{
		Namespace: p.Env.Spec.Namespace,
		Name:      p.Env.Name + "-minio",
	}

	secMap, err := deployMinio(
		p.Ctx,
		nn,
		p.Client,
		p.Env,
	)

	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(secMap["port"])

	m := &minioProvider{
		Ctx:       p.Ctx,
		AccessKey: secMap["accessKey"],
		SecretKey: secMap["secretKey"],
		Hostname:  secMap["hostname"],
		Port:      port,
	}

	endpoint := fmt.Sprintf("%v:%v", m.Hostname, m.Port)

	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(m.AccessKey, m.SecretKey, ""),
		Secure: false,
	})

	if err != nil {
		return m, errors.Wrap("Failed to create minio client", err)
	}

	m.Client = cl

	return m, nil
}

// DeployMinio creates the actual minio service to be used by clowdapps, this
// does not create buckets
func deployMinio(ctx context.Context, nn types.NamespacedName, client client.Client, env *crd.ClowdEnvironment) (map[string]string, error) {
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

	return *secMap, nil
}
