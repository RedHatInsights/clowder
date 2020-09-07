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

	"fmt"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"

	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//ObjectStoreMaker makes the StorageConfig object
type ObjectStoreMaker struct {
	*Maker
	config config.ObjectStoreConfig
}

//Make function for the StorageMaker
func (obs *ObjectStoreMaker) Make() error {
	obs.config = config.ObjectStoreConfig{}

	var f func() error

	switch obs.Base.Spec.ObjectStore.Provider {
	case "minio":
		f = obs.minio
	case "app-interface":
		f = obs.appInterface
	}

	if f != nil {
		return f()
	}

	return nil
}

//ApplyConfig for the StorageMaker
func (obs *ObjectStoreMaker) ApplyConfig(c *config.AppConfig) {
	c.ObjectStore = obs.config
}

func (obs *ObjectStoreMaker) appInterface() error {
	return nil
}

func (obs *ObjectStoreMaker) minio() error {
	if !obs.App.Spec.ObjectStore {
		return nil
	}

	minioObjName := fmt.Sprintf("%v-minio", obs.App.Name)
	minioNamespacedName := types.NamespacedName{
		Namespace: obs.App.Namespace,
		Name:      minioObjName,
	}

	dd := apps.Deployment{}
	err := obs.Client.Get(obs.Ctx, minioNamespacedName, &dd)

	update, err := updateOrErr(err)

	if err != nil {
		return err
	}

	dd.SetName(minioNamespacedName.Name)
	dd.SetNamespace(minioNamespacedName.Namespace)
	dd.SetLabels(obs.App.GetLabels())
	dd.SetOwnerReferences([]metav1.OwnerReference{obs.App.MakeOwnerReference()})

	dd.Spec.Replicas = obs.App.Spec.MinReplicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: obs.App.GetLabels()}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: minioNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: minioNamespacedName.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = obs.App.GetLabels()

	pullSecretRef := core.LocalObjectReference{Name: "quay-cloudservices-pull"}
	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{pullSecretRef}

	accessKey := core.EnvVar{Name: "MINIO_ACCESS_KEY", Value: utils.RandString(12)}
	secretKey := core.EnvVar{Name: "MINIO_SECRET_KEY", Value: utils.RandString(12)}
	envVars := []core.EnvVar{accessKey, secretKey}
	ports := []core.ContainerPort{
		{
			Name:          "minio",
			ContainerPort: 9000,
		},
	}

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

	if err = update.Apply(obs.Ctx, obs.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	err = obs.Client.Get(obs.Ctx, minioNamespacedName, &s)

	update, err = updateOrErr(err)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{}
	minioPort := core.ServicePort{Name: "minio", Port: 9000, Protocol: "TCP"}
	servicePorts = append(servicePorts, minioPort)

	obs.App.SetObjectMeta(&s, crd.Name(minioNamespacedName.Name), crd.Namespace(minioNamespacedName.Namespace))
	s.Spec.Selector = obs.App.GetLabels()
	s.Spec.Ports = servicePorts

	if err = update.Apply(obs.Ctx, obs.Client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}

	err = obs.Client.Get(obs.Ctx, minioNamespacedName, &pvc)

	update, err = updateOrErr(err)
	if err != nil {
		return err
	}

	pvc.SetName(minioNamespacedName.Name)
	pvc.SetNamespace(minioNamespacedName.Namespace)
	pvc.SetLabels(obs.App.GetLabels())
	pvc.SetOwnerReferences([]metav1.OwnerReference{obs.App.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(obs.Ctx, obs.Client, &pvc); err != nil {
		return err
	}

	obs.config.AccessKey = accessKey.Value
	obs.config.SecretKey = secretKey.Value

	return nil
}
