package v1alpha1

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.Validator[*ClowdApp] = &ClowdApp{}

func TestClowdAppAdmissionValidation(t *testing.T) {
	ctx := context.Background()
	validator := &ClowdApp{}
	valid := &ClowdApp{}
	invalid := &ClowdApp{}
	invalid.Spec.Database.Name = "database"
	invalid.Spec.Database.SharedDBAppName = "shared-database"

	if _, err := validator.ValidateCreate(ctx, invalid); !apierrors.IsInvalid(err) {
		t.Fatalf("create should reject conflicting database settings, got %v", err)
	}
	if _, err := validator.ValidateUpdate(ctx, valid, invalid); !apierrors.IsInvalid(err) {
		t.Fatalf("update should validate the new object, got %v", err)
	}
	if _, err := validator.ValidateCreate(ctx, valid); err != nil {
		t.Fatalf("create should accept valid settings, got %v", err)
	}
	if _, err := validator.ValidateUpdate(ctx, invalid, valid); err != nil {
		t.Fatalf("update should accept corrected settings, got %v", err)
	}
}
