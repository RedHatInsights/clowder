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
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var clowdapplog = logf.Log.WithName("clowdapp-resource")

// clowdAppValidator is a webhook that validates ClowdApp resources
type clowdAppValidator struct {
	client.Client
}

func (r *ClowdApp) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Add index for spec.envName field for webhook queries
	if err := mgr.GetFieldIndexer().IndexField(
		context.TODO(), &ClowdApp{}, "spec.envName", func(o client.Object) []string {
			return []string{o.(*ClowdApp).Spec.EnvName}
		}); err != nil {
		return err
	}

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&clowdAppValidator{Client: mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-cloud-redhat-com-v1alpha1-clowdapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=cloud.redhat.com,resources=clowdapps,verbs=create;update,versions=v1alpha1,name=vclowdapp.kb.io,admissionReviewVersions={v1}
//+kubebuilder:webhook:path=/mutate-pod,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=vclowdmutatepod.kb.io,admissionReviewVersions={v1}

// Define default validations that should always run
var defaultValidations = []appValidationFunc{
	validateDatabase,
	validateSidecars,
	validateInit,
	validateDeploymentStrategy,
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *clowdAppValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	clowdapp := obj.(*ClowdApp)
	clowdapplog.Info("validate create", "name", clowdapp.Name)

	// Create validations list with default validations plus duplicate name check
	validations := append([]appValidationFunc{v.validateDuplicateName}, defaultValidations...)

	return []string{}, v.processValidations(ctx, clowdapp, validations...)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *clowdAppValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	clowdapp := newObj.(*ClowdApp)
	oldClowdApp := oldObj.(*ClowdApp)
	clowdapplog.Info("validate update", "name", clowdapp.Name)

	// Start with default validations
	validations := make([]appValidationFunc, len(defaultValidations))
	copy(validations, defaultValidations)

	// Append duplicate name validation if names differ
	if oldClowdApp.Name != clowdapp.Name {
		validations = append(validations, v.validateDuplicateName)
	}

	return []string{}, v.processValidations(ctx, clowdapp, validations...)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *clowdAppValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	clowdapp := obj.(*ClowdApp)
	clowdapplog.Info("validate delete", "name", clowdapp.Name)
	return []string{}, nil
}

type appValidationFunc func(context.Context, client.Client, *ClowdApp) field.ErrorList

func (v *clowdAppValidator) processValidations(ctx context.Context, o *ClowdApp, vfns ...appValidationFunc) error {
	var allErrs field.ErrorList

	for _, validation := range vfns {
		if validation != nil {
			fieldList := validation(ctx, v.Client, o)
			if fieldList != nil {
				allErrs = append(allErrs, fieldList...)
			}
		}
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "cloud.redhat.com", Kind: "ClowdApp"},
		o.Name, allErrs,
	)
}

func validateDatabase(_ context.Context, _ client.Client, r *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}

	if r.Spec.Database.Name != "" && r.Spec.Database.SharedDBAppName != "" {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec.Database.Name", "spec.Database.SharedDBAppName"), "cannot set db name and sharedDbApp Name together"),
		)
	}

	if r.Spec.Database.SharedDBAppName != "" && r.Spec.Cyndi.Enabled {
		allErrs = append(allErrs, field.Forbidden(
			field.NewPath("spec.Database.SharedDBAppName", "spec.Cyndi.Enabled"), "cannot use cyndi with a shared database"),
		)
	}

	return allErrs
}

func validateInit(_ context.Context, _ client.Client, r *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}

	for depIdx, deployment := range r.Spec.Deployments {
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

func validateSidecars(_ context.Context, _ client.Client, r *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}
	for depIndx, deployment := range r.Spec.Deployments {
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
	for jobIndx, job := range r.Spec.Jobs {
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

func validateDeploymentStrategy(_ context.Context, _ client.Client, r *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}
	for depIndex, deployment := range r.Spec.Deployments {
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

func (v *clowdAppValidator) validateDuplicateName(ctx context.Context, c client.Client, r *ClowdApp) field.ErrorList {
	allErrs := field.ErrorList{}

	// Check if another ClowdApp with the same name already exists in the same ClowdEnvironment
	existingClowdApps := &ClowdAppList{}
	err := c.List(ctx, existingClowdApps, client.MatchingFields{
		"spec.envName": r.Spec.EnvName,
	})

	if err != nil {
		// If we got an error, log it but don't fail validation
		// This allows the webhook to continue functioning even if there are temporary
		// API server issues
		clowdapplog.Error(err, "Error checking for duplicate ClowdApp name", "name", r.Name)
		return allErrs
	}

	// Iterate through existing ClowdApps to check for duplicates with same name in different namespaces
	for _, existingApp := range existingClowdApps.Items {
		if existingApp.Name == r.Name && existingApp.Namespace != r.Namespace {
			allErrs = append(allErrs, field.Duplicate(
				field.NewPath("metadata").Child("name"),
				fmt.Sprintf("ClowdApp with name '%s' already exists in ClowdEnvironment '%s' in namespace '%s'", r.Name, r.Spec.EnvName, existingApp.Namespace)),
			)
		}
	}

	return allErrs
}
