// Package web provides web service and ingress management functionality for Clowder applications
package web

import (
	"encoding/json"

	crccaddy "github.com/RedHatInsights/crc-caddy-plugin"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	caddyreverseproxy "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
)

// ProxyRoute represents a proxy route configuration with upstream and path information
type ProxyRoute struct {
	Upstream string `json:"upstream"`
	Path     string `json:"path"`
}

// GenerateRoute creates a Caddy HTTP route configuration for the given proxy route
func GenerateRoute(upstream ProxyRoute, warnings *[]caddyconfig.Warning) *caddyhttp.Route {
	reverseProxy := caddyreverseproxy.Handler{
		Upstreams: []*caddyreverseproxy.Upstream{{
			Dial: upstream.Upstream,
		}},
	}

	routings := caddyhttp.Subroute{
		Routes: caddyhttp.RouteList{{
			HandlersRaw: []json.RawMessage{
				caddyconfig.JSONModuleObject(reverseProxy, "handler", "reverse_proxy", warnings),
			},
		}},
	}

	path := caddyhttp.MatchPath{upstream.Path}

	route := caddyhttp.Route{
		Group: "group2",
		HandlersRaw: []json.RawMessage{
			caddyconfig.JSONModuleObject(routings, "handler", "subroute", warnings),
		},
		MatcherSetsRaw: caddyhttp.RawMatcherSets{
			caddy.ModuleMap{"path": caddyconfig.JSON(path, warnings)},
		},
	}

	return &route
}

// GenerateConfig creates a complete Caddy configuration for the provided routes and TLS settings
func GenerateConfig(hostname string, bopAddress string, whitelist []string, appRoutes []ProxyRoute) (string, error) {
	var warnings []caddyconfig.Warning

	host := caddyhttp.MatchHost{hostname}

	crcauth := crccaddy.Middleware{
		Output:    "stdout",
		BOP:       bopAddress,
		Whitelist: whitelist,
	}

	subRoute := caddyhttp.Subroute{
		Routes: caddyhttp.RouteList{
			{
				HandlersRaw: []json.RawMessage{
					caddyconfig.JSONModuleObject(crcauth, "handler", "crcauth", &warnings),
				},
			},
		},
	}

	for _, appRoute := range appRoutes {
		subRoute.Routes = append(subRoute.Routes, *GenerateRoute(appRoute, &warnings))
	}

	sni := []string{hostname}

	caPool := caddytls.FileCAPool{
		TrustedCACertPEMFiles: []string{"/cas/ca.pem"},
	}
	appConfig := caddyhttp.App{
		HTTPPort:  8888,
		HTTPSPort: 9090,
		Servers: map[string]*caddyhttp.Server{"srv0": {
			Listen: []string{":9090"},
			Routes: []caddyhttp.Route{{
				Terminal: true,
				MatcherSetsRaw: caddyhttp.RawMatcherSets{
					caddy.ModuleMap{"host": caddyconfig.JSON(host, &warnings)},
				},
				Handlers: []caddyhttp.MiddlewareHandler{},
				//HandlersRaw: []json.RawMessage{caddyconfig.JSON(reverseProxy, &warnings)},nbu?
				HandlersRaw: []json.RawMessage{caddyconfig.JSONModuleObject(subRoute, "handler", "subroute", &warnings)},
			}},
			TLSConnPolicies: []*caddytls.ConnectionPolicy{{
				MatchersRaw: caddy.ModuleMap{"sni": caddyconfig.JSON(sni, &warnings)},
				CertSelection: &caddytls.CustomCertSelectionPolicy{
					AnyTag: []string{"cert0"},
				},
				ClientAuthentication: &caddytls.ClientAuthentication{
					Mode:  "verify_if_given",
					CARaw: caddyconfig.JSONModuleObject(caPool, "provider", "file", &warnings)},
			}, {}},
			Logs: &caddyhttp.ServerLogConfig{
				LoggerNames: map[string]caddyhttp.StringArray{"localhost.localdomain": {""}},
			},
		}},
	}

	fl := caddytls.FileLoader{{
		Certificate: "/certs/tls.crt",
		Key:         "/certs/tls.key",
		Tags:        []string{"cert0"},
	}}

	tlsConfig := caddytls.TLS{
		CertificatesRaw: caddy.ModuleMap{"load_files": caddyconfig.JSON(fl, &warnings)},
	}

	v := caddy.Config{
		StorageRaw: []byte{},
		AppsRaw: map[string]json.RawMessage{
			"http": caddyconfig.JSON(appConfig, &warnings),
			"tls":  caddyconfig.JSON(tlsConfig, &warnings),
		},
	}

	pretty, _ := json.MarshalIndent(v, "", "  ")
	return string(pretty) + "\n", nil
}
