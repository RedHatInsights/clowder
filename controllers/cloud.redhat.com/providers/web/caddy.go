package web

import (
	"encoding/json"

	crccaddy "github.com/RedHatInsights/crc-caddy-plugin"
	caddy "github.com/caddyserver/caddy/v2"
	caddyconfig "github.com/caddyserver/caddy/v2/caddyconfig"
	caddyhttp "github.com/caddyserver/caddy/v2/modules/caddyhttp"
	caddyreverseproxy "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	caddytls "github.com/caddyserver/caddy/v2/modules/caddytls"
)

type ProxyRoute struct {
	Upstream string `json:"upstream"`
	Path     string `json:"path"`
}

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

func GenerateConfig(hostname string, bopAddress string, whitelist []string, route []ProxyRoute) (string, error) {
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

	for _, route := range route {
		subRoute.Routes = append(subRoute.Routes, *GenerateRoute(route, &warnings))
	}

	sni := []string{hostname}

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
					Mode:                  "verify_if_given",
					TrustedCACertPEMFiles: []string{"/cas/ca.pem"},
				},
			}, {}},
			Logs: &caddyhttp.ServerLogConfig{
				LoggerNames: map[string]string{"localhost.localdomain:9090": ""},
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
	return string(pretty), nil
}
