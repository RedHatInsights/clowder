package sidecar

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
)

func TestGetOtelCollectorSidecar(t *testing.T) {
	tests := []struct {
		name        string
		env         *crd.ClowdEnvironment
		appSidecar  *crd.Sidecar
		expected    string
		description string
	}{
		{
			name: "app-level image override",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								Image: "env-level-image:v1.0.0",
							},
						},
					},
				},
			},
			appSidecar: &crd.Sidecar{
				Image: "app-level-image:v2.0.0",
			},
			expected:    "app-level-image:v2.0.0",
			description: "App-level image should take priority over environment-level image",
		},
		{
			name: "environment-level image fallback",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								Image: "env-level-image:v1.0.0",
							},
						},
					},
				},
			},
			appSidecar: &crd.Sidecar{
				Image: "", // No app-level image
			},
			expected:    "env-level-image:v1.0.0",
			description: "Environment-level image should be used when no app-level image is specified",
		},
		{
			name: "default image fallback",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								Image: "", // No env-level image
							},
						},
					},
				},
			},
			appSidecar: &crd.Sidecar{
				Image: "", // No app-level image
			},
			expected:    DefaultImageSideCarOtelCollector,
			description: "Default image should be used when no app-level or env-level image is specified",
		},
		{
			name: "nil app sidecar",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								Image: "env-level-image:v1.0.0",
							},
						},
					},
				},
			},
			appSidecar:  nil,
			expected:    "env-level-image:v1.0.0",
			description: "Environment-level image should be used when app sidecar is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOtelCollectorSidecar(tt.env, tt.appSidecar)
			if result != tt.expected {
				t.Errorf("GetOtelCollectorSidecar() = %v, expected %v. %s", result, tt.expected, tt.description)
			}
		})
	}
}

func TestGetOtelCollectorConfigMap(t *testing.T) {
	tests := []struct {
		name        string
		env         *crd.ClowdEnvironment
		appName     string
		appSidecar  *crd.Sidecar
		expected    string
		description string
	}{
		{
			name: "app-level configMap override",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								ConfigMap: "env-shared-config",
							},
						},
					},
				},
			},
			appName: "test-app",
			appSidecar: &crd.Sidecar{
				ConfigMap: "app-custom-config",
			},
			expected:    "app-custom-config",
			description: "App-level configMap should take priority over environment-level configMap",
		},
		{
			name: "environment-level configMap fallback",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								ConfigMap: "env-shared-config",
							},
						},
					},
				},
			},
			appName: "test-app",
			appSidecar: &crd.Sidecar{
				ConfigMap: "", // No app-level configMap
			},
			expected:    "env-shared-config",
			description: "Environment-level configMap should be used when no app-level configMap is specified",
		},
		{
			name: "default configMap fallback",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								ConfigMap: "", // No env-level configMap
							},
						},
					},
				},
			},
			appName: "test-app",
			appSidecar: &crd.Sidecar{
				ConfigMap: "", // No app-level configMap
			},
			expected:    "test-app-otel-config",
			description: "Default app-generated configMap name should be used when no app-level or env-level configMap is specified",
		},
		{
			name: "nil app sidecar",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							OtelCollector: crd.OtelCollectorConfig{
								ConfigMap: "env-shared-config",
							},
						},
					},
				},
			},
			appName:     "test-app",
			appSidecar:  nil,
			expected:    "env-shared-config",
			description: "Environment-level configMap should be used when app sidecar is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOtelCollectorConfigMap(tt.env, tt.appName, tt.appSidecar)
			if result != tt.expected {
				t.Errorf("GetOtelCollectorConfigMap() = %v, expected %v. %s", result, tt.expected, tt.description)
			}
		})
	}
}

func TestGetTokenRefresherSidecar(t *testing.T) {
	tests := []struct {
		name        string
		env         *crd.ClowdEnvironment
		expected    string
		description string
	}{
		{
			name: "environment-level image",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							TokenRefresher: crd.TokenRefresherConfig{
								Image: "env-token-refresher:v1.0.0",
							},
						},
					},
				},
			},
			expected:    "env-token-refresher:v1.0.0",
			description: "Environment-level image should be used for token refresher",
		},
		{
			name: "default image fallback",
			env: &crd.ClowdEnvironment{
				Spec: crd.ClowdEnvironmentSpec{
					Providers: crd.ProvidersConfig{
						Sidecars: crd.Sidecars{
							TokenRefresher: crd.TokenRefresherConfig{
								Image: "", // No env-level image
							},
						},
					},
				},
			},
			expected:    DefaultImageSideCarTokenRefresher,
			description: "Default image should be used when no env-level image is specified for token refresher",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenRefresherSidecar(tt.env)
			if result != tt.expected {
				t.Errorf("GetTokenRefresherSidecar() = %v, expected %v. %s", result, tt.expected, tt.description)
			}
		})
	}
}
