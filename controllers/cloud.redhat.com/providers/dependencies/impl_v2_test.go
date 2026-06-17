package dependencies

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
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

func TestBuildV2EndpointMap(t *testing.T) {
	t.Run("Empty endpoints returns nil", func(t *testing.T) {
		endpoints := []config.DependencyEndpoint{}
		result := buildV2EndpointMap(endpoints)
		assert.Nil(t, result)
	})

	t.Run("Single endpoint with base port", func(t *testing.T) {
		endpoints := []config.DependencyEndpoint{
			{
				Name:     "processor",
				App:      "puptoo",
				Hostname: "puptoo-processor.test-ns.svc",
				Port:     8000,
			},
		}

		result := buildV2EndpointMap(endpoints)

		assert.NotNil(t, result)
		assert.Contains(t, result, "puptoo")
		assert.Contains(t, result["puptoo"], "processor")
		assert.Equal(t, "http://puptoo-processor.test-ns.svc:8000", result["puptoo"]["processor"].Uri)
		assert.False(t, result["puptoo"]["processor"].Authenticated)
		assert.Nil(t, result["puptoo"]["processor"].CaCertificate)
	})

	t.Run("Endpoint with TLS port - opinionated (TLS wins)", func(t *testing.T) {
		tlsPath := "/cdapp/certs/service-ca.crt"
		endpoints := []config.DependencyEndpoint{
			{
				Name:      "processor",
				App:       "puptoo",
				Hostname:  "puptoo-processor.test-ns.svc",
				Port:      8000,
				TlsPort:   utils.IntPtr(8443),
				TlsCAPath: &tlsPath,
			},
		}

		result := buildV2EndpointMap(endpoints)

		// Only one entry - TLS wins (opinionated)
		assert.Contains(t, result["puptoo"], "processor")
		assert.Equal(t, "https://puptoo-processor.test-ns.svc:8443", result["puptoo"]["processor"].Uri)
		assert.False(t, result["puptoo"]["processor"].Authenticated)
		assert.NotNil(t, result["puptoo"]["processor"].CaCertificate)
		assert.Equal(t, tlsPath, *result["puptoo"]["processor"].CaCertificate)

		// No separate TLS entry in opinionated mode
		assert.NotContains(t, result["puptoo"], "processor_tls")
	})

	t.Run("Endpoint with H2C ports - opinionated (H2C wins over plaintext)", func(t *testing.T) {
		endpoints := []config.DependencyEndpoint{
			{
				Name:     "processor",
				App:      "puptoo",
				Hostname: "puptoo-processor.test-ns.svc",
				Port:     8000,
				H2CPort:  utils.IntPtr(8080),
			},
		}

		result := buildV2EndpointMap(endpoints)

		// Only one entry - H2C wins (opinionated)
		assert.Contains(t, result["puptoo"], "processor")
		assert.Equal(t, "http://puptoo-processor.test-ns.svc:8080", result["puptoo"]["processor"].Uri)
		assert.False(t, result["puptoo"]["processor"].Authenticated)

		// No separate H2C entry in opinionated mode
		assert.NotContains(t, result["puptoo"], "processor_h2c")
	})

	t.Run("Endpoint with all port types - opinionated (TLS wins)", func(t *testing.T) {
		tlsPath := "/cdapp/certs/service-ca.crt"
		endpoints := []config.DependencyEndpoint{
			{
				Name:       "processor",
				App:        "puptoo",
				Hostname:   "puptoo-processor.test-ns.svc",
				Port:       8000,
				TlsPort:    utils.IntPtr(8443),
				H2CPort:    utils.IntPtr(8080),
				H2CTLSPort: utils.IntPtr(8444),
				TlsCAPath:  &tlsPath,
			},
		}

		result := buildV2EndpointMap(endpoints)

		// Only one entry - TLS wins (highest priority in opinionated mode)
		assert.Len(t, result["puptoo"], 1, "Should have 1 entry in opinionated mode")
		assert.Contains(t, result["puptoo"], "processor")
		assert.Equal(t, "https://puptoo-processor.test-ns.svc:8443", result["puptoo"]["processor"].Uri)
		assert.False(t, result["puptoo"]["processor"].Authenticated)
		assert.NotNil(t, result["puptoo"]["processor"].CaCertificate)
		assert.Equal(t, tlsPath, *result["puptoo"]["processor"].CaCertificate)
	})

	t.Run("Multiple apps and services", func(t *testing.T) {
		endpoints := []config.DependencyEndpoint{
			{
				Name:     "processor",
				App:      "app1",
				Hostname: "app1-processor.ns.svc",
				Port:     8000,
			},
			{
				Name:     "api",
				App:      "app1",
				Hostname: "app1-api.ns.svc",
				Port:     8001,
			},
			{
				Name:     "service",
				App:      "app2",
				Hostname: "app2-service.ns.svc",
				Port:     9000,
			},
		}

		result := buildV2EndpointMap(endpoints)

		assert.Len(t, result, 2, "Should have 2 apps")
		assert.Len(t, result["app1"], 2, "app1 should have 2 services")
		assert.Len(t, result["app2"], 1, "app2 should have 1 service")
	})

	t.Run("Skip zero ports - opinionated (TLS wins)", func(t *testing.T) {
		endpoints := []config.DependencyEndpoint{
			{
				Name:     "processor",
				App:      "puptoo",
				Hostname: "puptoo-processor.test-ns.svc",
				Port:     0, // Zero port should be skipped
				TlsPort:  utils.IntPtr(8443),
			},
		}

		result := buildV2EndpointMap(endpoints)

		// Should have TLS entry (opinionated - single entry)
		assert.Contains(t, result["puptoo"], "processor")
		assert.Equal(t, "https://puptoo-processor.test-ns.svc:8443", result["puptoo"]["processor"].Uri)
		assert.False(t, result["puptoo"]["processor"].Authenticated)
	})
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
