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

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"
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

	switch obs.Base.Spec.ObjectStore.Provider {
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

func (obs *ObjectStoreMaker) minio() error {
	if obs.App.Spec.ObjectStore != nil {
		endpoint := obs.Base.GetAnnotations()["endpoint"]
		accessKeyID := obs.Base.GetAnnotations()["accessKey"]
		secretAccessKey := obs.Base.GetAnnotations()["secretKey"]
		// Initialize minio client object.
		minioClient, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
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
			if !found {
				err := minioClient.MakeBucket(obs.Ctx, bucket, minio.MakeBucketOptions{})
				if err != nil {
					return err
				}
			}
		}

		obs.config.AccessKey = accessKeyID
		obs.config.SecretKey = secretAccessKey
		obs.config.Endpoint = endpoint
	}

	return nil
}

// MakeMinio creates the actual minio service to be used by applications, this does not create buckets
func MakeMinio(client client.Client, ctx context.Context, req ctrl.Request, base *crd.InsightsBase) error {
	minioObjName := fmt.Sprintf("%v-minio", req.Name)
	minioNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      minioObjName,
	}

	dd := apps.Deployment{}
	err := client.Get(ctx, minioNamespacedName, &dd)
	update, err := utils.UpdateOrErr(err)

	if err != nil {
		return err
	}

	labels := base.GetLabels()
	labels["base-app"] = minioNamespacedName.Name

	dd.SetName(minioNamespacedName.Name)
	dd.SetNamespace(minioNamespacedName.Namespace)
	dd.SetLabels(labels)
	dd.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: minioNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: minioNamespacedName.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = labels

	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{{
		Name: "quay-cloudservices-pull",
	}}

	annotations := base.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{
			"accessKey": utils.RandString(12),
			"secretKey": utils.RandString(12),
			"endpoint":  fmt.Sprintf("%v.%v.svc:9000", minioObjName, req.Namespace),
		}
		base.SetAnnotations(annotations)
		if err = client.Update(ctx, base); err != nil {
			return err
		}
	}

	envVars := []core.EnvVar{
		{Name: "MINIO_ACCESS_KEY", Value: annotations["accessKey"]},
		{Name: "MINIO_SECRET_KEY", Value: annotations["secretKey"]},
	}

	ports := []core.ContainerPort{{
		Name:          "minio",
		ContainerPort: 9000,
	}}

	// TODO Readiness and Liveness probes

	c := core.Container{
		Name:  minioNamespacedName.Name,
		Image: "minio/minio",
		Env:   envVars,
		Ports: ports,
		VolumeMounts: []core.VolumeMount{{
			Name:      minioNamespacedName.Name,
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
		return err
	}

	s := core.Service{}
	err = client.Get(ctx, minioNamespacedName, &s)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{{
		Name:     "minio",
		Port:     9000,
		Protocol: "TCP",
	}}

	s.SetName(minioNamespacedName.Name)
	s.SetNamespace(minioNamespacedName.Namespace)
	s.SetLabels(labels)
	s.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	s.Spec.Selector = labels
	s.Spec.Ports = servicePorts

	if err = update.Apply(ctx, client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}

	err = client.Get(ctx, minioNamespacedName, &pvc)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	pvc.SetName(minioNamespacedName.Name)
	pvc.SetNamespace(minioNamespacedName.Namespace)
	pvc.SetLabels(labels)
	pvc.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(ctx, client, &pvc); err != nil {
		return err
	}
	return nil
}
