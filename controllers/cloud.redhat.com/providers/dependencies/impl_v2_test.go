package dependencies

import (
	"testing"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	"github.com/stretchr/testify/assert"
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
		assert.Nil(t, result["puptoo"]["processor"].CaCertificate)
	})

	t.Run("Endpoint with TLS port", func(t *testing.T) {
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

		// Base port entry
		assert.Contains(t, result["puptoo"], "processor")
		assert.Equal(t, "http://puptoo-processor.test-ns.svc:8000", result["puptoo"]["processor"].Uri)

		// TLS port entry
		assert.Contains(t, result["puptoo"], "processor_tls")
		assert.Equal(t, "https://puptoo-processor.test-ns.svc:8443", result["puptoo"]["processor_tls"].Uri)
		assert.NotNil(t, result["puptoo"]["processor_tls"].CaCertificate)
		assert.Equal(t, tlsPath, *result["puptoo"]["processor_tls"].CaCertificate)
	})

	t.Run("Endpoint with H2C ports", func(t *testing.T) {
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

		assert.Contains(t, result["puptoo"], "processor")
		assert.Contains(t, result["puptoo"], "processor_h2c")
		assert.Equal(t, "http://puptoo-processor.test-ns.svc:8080", result["puptoo"]["processor_h2c"].Uri)
	})

	t.Run("Endpoint with all port types", func(t *testing.T) {
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

		assert.Len(t, result["puptoo"], 4, "Should have 4 entries: base, tls, h2c, h2c_tls")
		assert.Contains(t, result["puptoo"], "processor")
		assert.Contains(t, result["puptoo"], "processor_tls")
		assert.Contains(t, result["puptoo"], "processor_h2c")
		assert.Contains(t, result["puptoo"], "processor_h2c_tls")

		// Verify H2C TLS has certificate
		assert.NotNil(t, result["puptoo"]["processor_h2c_tls"].CaCertificate)
		assert.Equal(t, tlsPath, *result["puptoo"]["processor_h2c_tls"].CaCertificate)
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

	t.Run("Skip zero ports", func(t *testing.T) {
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

		// Should only have TLS entry, not base port
		assert.NotContains(t, result["puptoo"], "processor", "Should skip zero port")
		assert.Contains(t, result["puptoo"], "processor_tls")
	})
}

func TestMakeV2DependencyEndpoints(t *testing.T) {
	t.Run("Empty endpoints does nothing", func(t *testing.T) {
		cfg := &config.AppConfig{
			Endpoints: []config.DependencyEndpoint{},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
			},
		}

		dep.makeV2DependencyEndpoints()

		assert.Nil(t, cfg.DependencyEndpoints)
	})

	t.Run("Populates V2 structure from V1 endpoints", func(t *testing.T) {
		tlsPath := "/cdapp/certs/service-ca.crt"
		cfg := &config.AppConfig{
			Endpoints: []config.DependencyEndpoint{
				{
					Name:      "processor",
					App:       "puptoo",
					Hostname:  "puptoo-processor.test-ns.svc",
					Port:      8000,
					TlsPort:   utils.IntPtr(8443),
					TlsCAPath: &tlsPath,
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
			},
		}

		dep.makeV2DependencyEndpoints()

		assert.NotNil(t, cfg.DependencyEndpoints)
		assert.NotNil(t, cfg.DependencyEndpoints.V2)

		// Check structure - V2 is map[string]interface{}
		assert.Contains(t, cfg.DependencyEndpoints.V2, "puptoo")

		// Get app endpoints
		appEndpoints, ok := cfg.DependencyEndpoints.V2["puptoo"].(map[string]config.DependencyEndpointV2)
		assert.True(t, ok, "App endpoints should be map[string]DependencyEndpointV2")
		assert.Contains(t, appEndpoints, "processor")
		assert.Equal(t, "http://puptoo-processor.test-ns.svc:8000", appEndpoints["processor"].Uri)
	})
}

func TestMakeV2PrivateDependencyEndpoints(t *testing.T) {
	t.Run("Empty private endpoints does nothing", func(t *testing.T) {
		cfg := &config.AppConfig{
			PrivateEndpoints: []config.PrivateDependencyEndpoint{},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
			},
		}

		dep.makeV2PrivateDependencyEndpoints()

		assert.Nil(t, cfg.PrivateDependencyEndpoints)
	})

	t.Run("Populates V2 private structure from V1 private endpoints", func(t *testing.T) {
		tlsPath := "/cdapp/certs/service-ca.crt"
		cfg := &config.AppConfig{
			PrivateEndpoints: []config.PrivateDependencyEndpoint{
				{
					Name:      "processor",
					App:       "puptoo",
					Hostname:  "puptoo-processor.test-ns.svc",
					Port:      10000,
					TlsPort:   utils.IntPtr(10443),
					TlsCAPath: &tlsPath,
				},
			},
		}
		dep := &dependenciesProvider{
			Provider: providers.Provider{
				Config: cfg,
			},
		}

		dep.makeV2PrivateDependencyEndpoints()

		assert.NotNil(t, cfg.PrivateDependencyEndpoints)
		assert.NotNil(t, cfg.PrivateDependencyEndpoints.V2)

		// Check structure - V2 is map[string]interface{}
		assert.Contains(t, cfg.PrivateDependencyEndpoints.V2, "puptoo")

		// Get app endpoints
		appEndpoints, ok := cfg.PrivateDependencyEndpoints.V2["puptoo"].(map[string]config.DependencyEndpointV2)
		assert.True(t, ok, "App endpoints should be map[string]DependencyEndpointV2")
		assert.Contains(t, appEndpoints, "processor")
		assert.Equal(t, "http://puptoo-processor.test-ns.svc:10000", appEndpoints["processor"].Uri)
	})
}
