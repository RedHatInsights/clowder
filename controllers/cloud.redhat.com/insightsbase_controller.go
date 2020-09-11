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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"
)

// InsightsBaseReconciler reconciles a InsightsBase object
type InsightsBaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsbases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.redhat.com,resources=insightsbases/status,verbs=get;update;patch

func (r *InsightsBaseReconciler) makeMinio(ctx context.Context, req ctrl.Request, base *crd.InsightsBase) error {
	minioObjName := fmt.Sprintf("%v-minio", req.Name)
	minioNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      minioObjName,
	}

	dd := apps.Deployment{}
	err := r.Client.Get(ctx, minioNamespacedName, &dd)
	update, err := utils.UpdateOrErr(err)

	if err != nil {
		return err
	}

	dd.SetName(minioNamespacedName.Name)
	dd.SetNamespace(minioNamespacedName.Namespace)
	dd.SetLabels(base.GetLabels())
	dd.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: base.GetLabels()}
	dd.Spec.Template.Spec.Volumes = []core.Volume{{
		Name: minioNamespacedName.Name,
		VolumeSource: core.VolumeSource{
			PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
				ClaimName: minioNamespacedName.Name,
			},
		}},
	}
	dd.Spec.Template.ObjectMeta.Labels = base.GetLabels()

	pullSecretRef := core.LocalObjectReference{Name: "quay-cloudservices-pull"}
	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{pullSecretRef}

	var accessKey, secretKey core.EnvVar
	if !update {
		accessKey = core.EnvVar{Name: "MINIO_ACCESS_KEY", Value: utils.RandString(12)}
		secretKey = core.EnvVar{Name: "MINIO_SECRET_KEY", Value: utils.RandString(12)}
		annotations := base.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["accessKey"] = accessKey.Value
		annotations["secretKey"] = secretKey.Value
		annotations["endpoint"] = fmt.Sprintf("%v.%v.svc:9000", minioObjName, req.Namespace)
		base.SetAnnotations(annotations)
		if err = r.Client.Update(ctx, base); err != nil {
			return err
		}
	} else {
		appConfig := base.GetAnnotations()
		if err != nil {
			return err
		}
		accessKey = core.EnvVar{Name: "MINIO_ACCESS_KEY", Value: appConfig["accessKey"]}
		secretKey = core.EnvVar{Name: "MINIO_SECRET_KEY", Value: appConfig["secretKey"]}
	}
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
	dd.Spec.Template.SetLabels(base.GetLabels())

	if err = update.Apply(ctx, r.Client, &dd); err != nil {
		return err
	}

	s := core.Service{}
	err = r.Client.Get(ctx, minioNamespacedName, &s)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	servicePorts := []core.ServicePort{}
	minioPort := core.ServicePort{Name: "minio", Port: 9000, Protocol: "TCP"}
	servicePorts = append(servicePorts, minioPort)

	s.SetName(minioNamespacedName.Name)
	s.SetNamespace(minioNamespacedName.Namespace)
	s.SetLabels(base.GetLabels())
	s.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})

	s.Spec.Selector = base.GetLabels()
	s.Spec.Ports = servicePorts

	if err = update.Apply(ctx, r.Client, &s); err != nil {
		return err
	}

	pvc := core.PersistentVolumeClaim{}

	err = r.Client.Get(ctx, minioNamespacedName, &pvc)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	pvc.SetName(minioNamespacedName.Name)
	pvc.SetNamespace(minioNamespacedName.Namespace)
	pvc.SetLabels(base.GetLabels())
	pvc.SetOwnerReferences([]metav1.OwnerReference{base.MakeOwnerReference()})
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse("1Gi"),
		},
	}

	if err = update.Apply(ctx, r.Client, &pvc); err != nil {
		return err
	}
	return nil
}

//Reconcile fn
func (r *InsightsBaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("insightsbase", req.NamespacedName)

	base := crd.InsightsBase{}
	err := r.Client.Get(ctx, req.NamespacedName, &base)

	if err != nil {
		return ctrl.Result{}, err
	}

	if base.Spec.ObjectStore.Provider == "minio" {
		err = r.makeMinio(ctx, req, &base)

		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up with manager
func (r *InsightsBaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.InsightsBase{}).
		Complete(r)
}
