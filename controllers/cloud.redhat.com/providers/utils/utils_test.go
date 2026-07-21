package utils

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func boolPtr(b bool) *bool { return &b }

func TestIsPublicAuthenticated(t *testing.T) {
	tests := []struct {
		name                 string
		override             *bool
		defaultAuthenticated bool
		expected             bool
	}{
		{
			name:                 "nil override, default false (ClowdApp) returns false",
			override:             nil,
			defaultAuthenticated: false,
			expected:             false,
		},
		{
			name:                 "nil override, default true (ClowdAppRef) returns true",
			override:             nil,
			defaultAuthenticated: true,
			expected:             true,
		},
		{
			name:                 "override true with default false (ClowdApp opt-in) returns true",
			override:             boolPtr(true),
			defaultAuthenticated: false,
			expected:             true,
		},
		{
			name:                 "override false with default true (ClowdAppRef opt-out) returns false",
			override:             boolPtr(false),
			defaultAuthenticated: true,
			expected:             false,
		},
		{
			name:                 "override true with default true (ClowdAppRef explicit) returns true",
			override:             boolPtr(true),
			defaultAuthenticated: true,
			expected:             true,
		},
		{
			name:                 "override false with default false (ClowdApp explicit) returns false",
			override:             boolPtr(false),
			defaultAuthenticated: false,
			expected:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &crd.WebServices{
				Public: crd.PublicWebService{
					Authenticated: tt.override,
				},
			}
			result := IsPublicAuthenticated(ws, tt.defaultAuthenticated)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPrivateAuthenticated(t *testing.T) {
	tests := []struct {
		name                 string
		override             *bool
		defaultAuthenticated bool
		expected             bool
	}{
		{
			name:                 "nil override, default false (ClowdApp) returns false",
			override:             nil,
			defaultAuthenticated: false,
			expected:             false,
		},
		{
			name:                 "nil override, default true (ClowdAppRef) returns true",
			override:             nil,
			defaultAuthenticated: true,
			expected:             true,
		},
		{
			name:                 "override true with default false (ClowdApp opt-in) returns true",
			override:             boolPtr(true),
			defaultAuthenticated: false,
			expected:             true,
		},
		{
			name:                 "override false with default true (ClowdAppRef opt-out) returns false",
			override:             boolPtr(false),
			defaultAuthenticated: true,
			expected:             false,
		},
		{
			name:                 "override true with default true (ClowdAppRef explicit) returns true",
			override:             boolPtr(true),
			defaultAuthenticated: true,
			expected:             true,
		},
		{
			name:                 "override false with default false (ClowdApp explicit) returns false",
			override:             boolPtr(false),
			defaultAuthenticated: false,
			expected:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &crd.WebServices{
				Private: crd.PrivateWebService{
					Authenticated: tt.override,
				},
			}
			result := IsPrivateAuthenticated(ws, tt.defaultAuthenticated)
			assert.Equal(t, tt.expected, result)
		})
	}
}
