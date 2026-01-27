/*
Copyright 2021.

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

package v1alpha1

import (
	"context"
	"fmt"

	apps "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var clowdapplog = logf.Log.WithName("clowdapp-resource")

// SetupWebhookWithManager configures the webhook for this ClowdApp resource
func (i *ClowdApp) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(i).
		WithValidator(i).
		Complete()
}

//+kubebuilder:webhook:path=/validate-cloud-redhat-com-v1alpha1-clowdapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloud.redhat.com,resources=clowdapps,verbs=create;update,versions=v1alpha1,name=vclowdapp.kb.io,admissionReviewVersions={v1}
//+kubebuilder:webhook:path=/mutate-pod,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=vclowdmutatepod.kb.io,admissionReviewVersions={v1}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (i *ClowdApp) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	clowdApp, ok := obj.(*ClowdApp)
	if !ok {
		return nil, fmt.Errorf("expected ClowdApp but got %T", obj)
	}
	clowdapplog.Info("validate create", "name", clowdApp.Name)

	return []string{}, i.processValidations(clowdApp,
		validateDatabase,
		validateSidecars,
		validateInit,
		validateDeploymentStrategy,
	)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (i *ClowdApp) ValidateUpdate(_ context.Context, _ runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	clowdApp, ok := newObj.(*ClowdApp)
	if !ok {
		return nil, fmt.Errorf("expected ClowdApp but got %T", newObj)
	}
	clowdapplog.Info("validate update", "name", clowdApp.Name)

	return []string{}, i.processValidations(clowdApp,
		validateDatabase,
		validateSidecars,
		validateInit,
		validateDeploymentStrategy,
	)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (i *ClowdApp) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	clowdapplog.Info("validate delete", "name", i.Name)
	return []string{}, nil
}

type appValidationFunc func(*ClowdApp) field.ErrorList

func (i *ClowdApp) processValidations(o *ClowdApp, vfns ...appValidationFunc) error {
	var allErrs field.ErrorList

	for _, validation := range vfns {
		fieldList := validation(o)
		if fieldList != nil {
			allErrs = append(allErrs, fieldList...)
		}
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "cloud.redhat.com", Kind: "ClowdApp"},
		i.Name, allErrs,
	)
}

func validateDatabase(i *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}

	if i.Spec.Database.Name != "" && i.Spec.Database.SharedDBAppName != "" {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec.Database.Name", "spec.Database.SharedDBAppName"), "cannot set db name and sharedDbApp Name together"),
		)
	}

	if i.Spec.Database.SharedDBAppName != "" && i.Spec.Cyndi.Enabled {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec.Database.SharedDBAppName", "spec.Cyndi.Enabled"), "cannot use cyndi with a shared database"),
		)
	}

	return allErrs
}

func validateInit(i *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}

	for depIdx, deployment := range i.Spec.Deployments {
		if len(deployment.PodSpec.InitContainers) > 1 {
			for icIdx, ic := range deployment.PodSpec.InitContainers {
				if ic.Name == "" {
					allErrs = append(allErrs, field.Forbidden(
						field.NewPath(
							fmt.Sprintf("spec.Deployments[%d].PodSpec.InitContainers[%d]", depIdx, icIdx),
						), "multiple initcontainers must have a name"),
					)
				}
			}
		}
	}

	return allErrs
}

func validateSidecars(i *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}
	for depIndx, deployment := range i.Spec.Deployments {
		for carIndx, sidecar := range deployment.PodSpec.Sidecars {
			if sidecar.Name != "token-refresher" && sidecar.Name != "otel-collector" {
				allErrs = append(
					allErrs,
					field.Forbidden(
						field.NewPath(fmt.Sprintf("spec.Deployment[%d].Sidecars[%d]", depIndx, carIndx)),
						"Sidecar is of unknown type, must be one of [token-refresher] or [otel-collector]",
					),
				)
			}
		}
	}
	for jobIndx, job := range i.Spec.Jobs {
		if job.Schedule == "" {
			continue
		}
		for carIndx, sidecar := range job.PodSpec.Sidecars {
			if sidecar.Name != "token-refresher" && sidecar.Name != "otel-collector" {
				allErrs = append(
					allErrs,
					field.Forbidden(
						field.NewPath(fmt.Sprintf("spec.Deployment[%d].Sidecars[%d]", jobIndx, carIndx)),
						"Sidecar is of unknown type, must be one of [token-refresher] or [otel-collector]",
					),
				)
			}
		}
	}
	return allErrs
}

func validateDeploymentStrategy(i *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}
	for depIndex, deployment := range i.Spec.Deployments {
		if deployment.DeploymentStrategy != nil && deployment.WebServices.Public.Enabled && deployment.DeploymentStrategy.PrivateStrategy == apps.RecreateDeploymentStrategyType {
			allErrs = append(
				allErrs,
				field.Forbidden(
					field.NewPath(fmt.Sprintf("spec.Deployment[%d]", depIndex)),
					"privateStrategy cannot be set to recreate for public web enabled deployments",
				),
			)
		}
	}
	return allErrs
}
