package dependencies

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConstructEndpointURI(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		hostname string
		port     int
		expected string
	}{
		{
			name:     "HTTP URI",
			protocol: "http",
			hostname: "service.namespace.svc",
			port:     8000,
			expected: "http://service.namespace.svc:8000",
		},
		{
			name:     "HTTPS URI",
			protocol: "https",
			hostname: "service.namespace.svc",
			port:     8443,
			expected: "https://service.namespace.svc:8443",
		},
		{
			name:     "HTTP with different port",
			protocol: "http",
			hostname: "app-processor.test-ns.svc",
			port:     3000,
			expected: "http://app-processor.test-ns.svc:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constructEndpointURI(tt.protocol, tt.hostname, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMakeV2DependencyEndpoints(t *testing.T) {
	t.Run("Empty dependency list does nothing", func(t *testing.T) {
		cfg := &config.AppConfig{}
		env := &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Web: crd.WebConfig{
						Port: 8000,
					},
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
				Env:    env,
			},
		}

		dep.makeV2DependencyEndpoints(&crd.ClowdAppList{}, &crd.ClowdAppRefList{}, []string{}, "consumer")

		assert.Nil(t, cfg.DependencyEndpoints)
	})

	t.Run("Creates V2 endpoint for ClowdApp dependency (authenticated: false)", func(t *testing.T) {
		cfg := &config.AppConfig{}
		env := &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Web: crd.WebConfig{
						Port:        8000,
						PrivatePort: 10000,
						TLS: crd.TLS{
							Enabled:     true,
							Port:        8443,
							PrivatePort: 10443,
						},
					},
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
				Env:    env,
			},
		}

		apps := &crd.ClowdAppList{
			Items: []crd.ClowdApp{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "puptoo",
					Namespace: "test-ns",
				},
				Spec: crd.ClowdAppSpec{
					Deployments: []crd.Deployment{{
						Name: "processor",
						WebServices: crd.WebServices{
							Public: crd.PublicWebService{
								Enabled: true,
							},
						},
					}},
				},
			}},
		}

		dep.makeV2DependencyEndpoints(apps, &crd.ClowdAppRefList{}, []string{"puptoo"}, "consumer")

		assert.NotNil(t, cfg.DependencyEndpoints)
		assert.NotNil(t, cfg.DependencyEndpoints.V2)
		assert.Contains(t, cfg.DependencyEndpoints.V2, "puptoo")

		appEndpoints, ok := cfg.DependencyEndpoints.V2["puptoo"].(map[string]config.DependencyEndpointV2)
		assert.True(t, ok, "App endpoints should be map[string]DependencyEndpointV2")
		assert.Contains(t, appEndpoints, "processor")

		// TLS enabled, so should use HTTPS port
		assert.Equal(t, "https://puptoo-processor.test-ns.svc:8443", appEndpoints["processor"].Uri)
		assert.False(t, appEndpoints["processor"].Authenticated, "ClowdApp should have authenticated=false")
		assert.NotNil(t, appEndpoints["processor"].CaCertificate, "TLS endpoint should have CA certificate")
	})

	t.Run("Creates V2 endpoint for ClowdAppRef dependency (authenticated: true)", func(t *testing.T) {
		cfg := &config.AppConfig{}
		env := &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Web: crd.WebConfig{
						Port: 8000,
					},
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
				Env:    env,
			},
		}

		appRefs := &crd.ClowdAppRefList{
			Items: []crd.ClowdAppRef{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-service",
					Namespace: "test-ns",
				},
				Spec: crd.ClowdAppRefSpec{
					Deployments: []crd.ClowdAppRefDeployment{{
						Name:     "api",
						Hostname: "external-api.example.com",
						WebServices: crd.WebServices{
							Public: crd.PublicWebService{
								Enabled: true,
							},
						},
					}},
					RemoteEnvironment: crd.ClowdAppRefRemoteEnvironment{
						Port: 443,
						TLS: crd.TLS{
							Enabled: true,
							Port:    443,
						},
					},
				},
			}},
		}

		dep.makeV2DependencyEndpoints(&crd.ClowdAppList{}, appRefs, []string{"external-service"}, "consumer")

		assert.NotNil(t, cfg.DependencyEndpoints)
		assert.NotNil(t, cfg.DependencyEndpoints.V2)
		assert.Contains(t, cfg.DependencyEndpoints.V2, "external-service")

		appEndpoints, ok := cfg.DependencyEndpoints.V2["external-service"].(map[string]config.DependencyEndpointV2)
		assert.True(t, ok, "App endpoints should be map[string]DependencyEndpointV2")
		assert.Contains(t, appEndpoints, "api")

		assert.Equal(t, "https://external-api.example.com:443", appEndpoints["api"].Uri)
		assert.True(t, appEndpoints["api"].Authenticated, "ClowdAppRef should have authenticated=true")
		assert.Nil(t, appEndpoints["api"].CaCertificate, "ClowdAppRef should not have ca_certificate")
	})
}

func TestMakeV2PrivateDependencyEndpoints(t *testing.T) {
	t.Run("Empty dependency list does nothing", func(t *testing.T) {
		cfg := &config.AppConfig{}
		env := &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Web: crd.WebConfig{
						PrivatePort: 10000,
					},
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
				Env:    env,
			},
		}

		dep.makeV2PrivateDependencyEndpoints(&crd.ClowdAppList{}, &crd.ClowdAppRefList{}, []string{}, "consumer")

		assert.Nil(t, cfg.PrivateDependencyEndpoints)
	})

	t.Run("Creates V2 private endpoint for ClowdApp dependency (authenticated: false)", func(t *testing.T) {
		cfg := &config.AppConfig{}
		env := &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Web: crd.WebConfig{
						Port:        8000,
						PrivatePort: 10000,
						TLS: crd.TLS{
							Enabled:     true,
							Port:        8443,
							PrivatePort: 10443,
						},
					},
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
				Env:    env,
			},
		}

		apps := &crd.ClowdAppList{
			Items: []crd.ClowdApp{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "puptoo",
					Namespace: "test-ns",
				},
				Spec: crd.ClowdAppSpec{
					Deployments: []crd.Deployment{{
						Name: "processor",
						WebServices: crd.WebServices{
							Private: crd.PrivateWebService{
								Enabled: true,
							},
						},
					}},
				},
			}},
		}

		dep.makeV2PrivateDependencyEndpoints(apps, &crd.ClowdAppRefList{}, []string{"puptoo"}, "consumer")

		assert.NotNil(t, cfg.PrivateDependencyEndpoints)
		assert.NotNil(t, cfg.PrivateDependencyEndpoints.V2)
		assert.Contains(t, cfg.PrivateDependencyEndpoints.V2, "puptoo")

		appEndpoints, ok := cfg.PrivateDependencyEndpoints.V2["puptoo"].(map[string]config.DependencyEndpointV2)
		assert.True(t, ok, "App endpoints should be map[string]DependencyEndpointV2")
		assert.Contains(t, appEndpoints, "processor")

		// TLS enabled on private, so should use HTTPS private port
		assert.Equal(t, "https://puptoo-processor.test-ns.svc:10443", appEndpoints["processor"].Uri)
		assert.False(t, appEndpoints["processor"].Authenticated, "ClowdApp should have authenticated=false")
		assert.NotNil(t, appEndpoints["processor"].CaCertificate, "TLS endpoint should have CA certificate")
	})

	t.Run("Creates V2 private endpoint for ClowdAppRef dependency (authenticated: true)", func(t *testing.T) {
		cfg := &config.AppConfig{}
		env := &crd.ClowdEnvironment{
			Spec: crd.ClowdEnvironmentSpec{
				Providers: crd.ProvidersConfig{
					Web: crd.WebConfig{
						PrivatePort: 10000,
					},
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
				Env:    env,
			},
		}

		appRefs := &crd.ClowdAppRefList{
			Items: []crd.ClowdAppRef{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-service",
					Namespace: "test-ns",
				},
				Spec: crd.ClowdAppRefSpec{
					Deployments: []crd.ClowdAppRefDeployment{{
						Name:     "api",
						Hostname: "external-api.example.com",
						WebServices: crd.WebServices{
							Private: crd.PrivateWebService{
								Enabled: true,
							},
						},
					}},
					RemoteEnvironment: crd.ClowdAppRefRemoteEnvironment{
						PrivatePort: 10000,
					},
				},
			}},
		}

		dep.makeV2PrivateDependencyEndpoints(&crd.ClowdAppList{}, appRefs, []string{"external-service"}, "consumer")

		assert.NotNil(t, cfg.PrivateDependencyEndpoints)
		assert.NotNil(t, cfg.PrivateDependencyEndpoints.V2)
		assert.Contains(t, cfg.PrivateDependencyEndpoints.V2, "external-service")

		appEndpoints, ok := cfg.PrivateDependencyEndpoints.V2["external-service"].(map[string]config.DependencyEndpointV2)
		assert.True(t, ok, "App endpoints should be map[string]DependencyEndpointV2")
		assert.Contains(t, appEndpoints, "api")

		assert.Equal(t, "http://external-api.example.com:10000", appEndpoints["api"].Uri)
		assert.True(t, appEndpoints["api"].Authenticated, "ClowdAppRef should have authenticated=true")
		assert.Nil(t, appEndpoints["api"].CaCertificate, "ClowdAppRef should not have ca_certificate")
	})
}
