package sidecar

import (
	"fmt"
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	core "k8s.io/api/core/v1"
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

func TestGetOtelCollectorVolumeMounts(t *testing.T) {
	env := &crd.ClowdEnvironment{
		Spec: crd.ClowdEnvironmentSpec{
			Providers: crd.ProvidersConfig{
				Sidecars: crd.Sidecars{
					OtelCollector: crd.OtelCollectorConfig{
						Enabled: true,
					},
				},
			},
		},
	}
	appName := "test-app"
	expectedConfigMount := fmt.Sprintf("%s-otel-config", appName)

	t.Run("no custom volumeMounts", func(t *testing.T) {
		sidecar := &crd.Sidecar{Name: "otel-collector", Enabled: true}
		cont := getOtelCollector(appName, env, nil, sidecar)
		if len(cont.VolumeMounts) != 1 {
			t.Fatalf("expected 1 volumeMount, got %d", len(cont.VolumeMounts))
		}
		if cont.VolumeMounts[0].Name != expectedConfigMount {
			t.Errorf("expected volumeMount name %s, got %s", expectedConfigMount, cont.VolumeMounts[0].Name)
		}
	})

	t.Run("with custom volumeMounts", func(t *testing.T) {
		sidecar := &crd.Sidecar{
			Name:    "otel-collector",
			Enabled: true,
			VolumeMounts: []core.VolumeMount{
				{Name: "logs", MountPath: "/logs"},
				{Name: "data", MountPath: "/data", ReadOnly: true},
			},
		}
		cont := getOtelCollector(appName, env, nil, sidecar)
		if len(cont.VolumeMounts) != 3 {
			t.Fatalf("expected 3 volumeMounts, got %d", len(cont.VolumeMounts))
		}
		if cont.VolumeMounts[0].Name != expectedConfigMount {
			t.Errorf("first volumeMount should be config, got %s", cont.VolumeMounts[0].Name)
		}
		if cont.VolumeMounts[1].Name != "logs" || cont.VolumeMounts[1].MountPath != "/logs" {
			t.Errorf("second volumeMount should be logs:/logs, got %s:%s", cont.VolumeMounts[1].Name, cont.VolumeMounts[1].MountPath)
		}
		if cont.VolumeMounts[2].Name != "data" || cont.VolumeMounts[2].MountPath != "/data" || !cont.VolumeMounts[2].ReadOnly {
			t.Errorf("third volumeMount should be data:/data (readOnly), got %s:%s (readOnly=%v)", cont.VolumeMounts[2].Name, cont.VolumeMounts[2].MountPath, cont.VolumeMounts[2].ReadOnly)
		}
	})

	t.Run("nil sidecar preserves default mount", func(t *testing.T) {
		cont := getOtelCollector(appName, env, nil, nil)
		if len(cont.VolumeMounts) != 1 {
			t.Fatalf("expected 1 volumeMount, got %d", len(cont.VolumeMounts))
		}
		if cont.VolumeMounts[0].Name != expectedConfigMount {
			t.Errorf("expected volumeMount name %s, got %s", expectedConfigMount, cont.VolumeMounts[0].Name)
		}
	})

	t.Run("empty volumeMounts slice", func(t *testing.T) {
		sidecar := &crd.Sidecar{
			Name:         "otel-collector",
			Enabled:      true,
			VolumeMounts: []core.VolumeMount{},
		}
		cont := getOtelCollector(appName, env, nil, sidecar)
		if len(cont.VolumeMounts) != 1 {
			t.Fatalf("expected 1 volumeMount, got %d", len(cont.VolumeMounts))
		}
	})
}
