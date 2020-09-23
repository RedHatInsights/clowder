/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package makers

import (
	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date

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

//ObjectStoreMaker makes the StorageConfig object
type ObjectStoreMaker struct {
	*Maker
	config config.ObjectStoreConfig
}

//Make function for the StorageMaker
func (obs *ObjectStoreMaker) Make() (ctrl.Result, error) {
	obs.config = config.ObjectStoreConfig{}

	var f func() error

	switch obs.Env.Spec.ObjectStore.Provider {
	case "minio":
		f = obs.minio
	case "app-interface":
		f = obs.appInterface
	}

	if f != nil {
		return ctrl.Result{}, f()
	}

	return ctrl.Result{}, nil
}

//ApplyConfig for the StorageMaker
func (obs *ObjectStoreMaker) ApplyConfig(c *config.AppConfig) {
	c.ObjectStore = &obs.config
}

func (obs *ObjectStoreMaker) appInterface() error {
	return nil
}

func configFromEnv(env *crd.Environment, c client.Client) (*config.ObjectStoreConfig, error) {
	conf := &config.ObjectStoreConfig{
		Hostname: env.Status.ObjectStore.Minio.Hostname,
		Port:     int(env.Status.ObjectStore.Minio.Port),
	}

	secretName := env.Status.ObjectStore.Minio.Credentials
	name := types.NamespacedName{
		Name:      secretName.Name,
		Namespace: secretName.Namespace,
	}
	secret := core.Secret{}
	err := c.Get(context.Background(), name, &secret)

	if err != nil {
		return conf, err
	}

	conf.AccessKey = string(secret.Data["accessKey"])
	conf.SecretKey = string(secret.Data["secretKey"])

	return conf, nil
}

func (obs *ObjectStoreMaker) minio() error {
	if obs.App.Spec.ObjectStore != nil {
		c, err := configFromEnv(obs.Env, obs.Client)

		obs.config = *c

		if err != nil {
			return err
		}
		endpoint := fmt.Sprintf("%v:%v", obs.config.Hostname, obs.config.Port)
		// Initialize minio client object.
		minioClient, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(obs.config.AccessKey, obs.config.SecretKey, ""),
			Secure: false,
		})
		if err != nil {
			return err
		}

		for _, bucket := range obs.App.Spec.ObjectStore {
			found, err := minioClient.BucketExists(obs.Ctx, bucket)
			if err != nil {
				return err
			}
			if found {
				continue
			}

			if err := minioClient.MakeBucket(obs.Ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return err
			}
		}

	}

	return nil
}

// MakeMinio creates the actual minio service to be used by applications, this does not create buckets
func MakeMinio(m *Maker) (ctrl.Result, error) {
	result := ctrl.Result{}
	nn := GetNamespacedName(m.Request, "%v-minio", m.Env.Spec.Namespace)

	dd := apps.Deployment{}
	update, err := m.Get(nn, &dd)
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
