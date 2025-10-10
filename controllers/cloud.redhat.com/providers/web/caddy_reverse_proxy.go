package web

import (
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
)

func generateServers(pub bool, priv bool, pubPort int32, privPort int32, appPubPort int32, appPrivPort int32, protocol string) (map[string]*caddyhttp.Server, error) {
	servers := make(map[string]*caddyhttp.Server)

	tlsConnPolicy := []*caddytls.ConnectionPolicy{{
		CertSelection: &caddytls.CustomCertSelectionPolicy{
			AnyTag: []string{"cert0"},
		},
	}}

	if pub {
		pubServer := generateServer(pubPort, appPubPort, tlsConnPolicy, protocol)
		servers["pubServer"] = pubServer
	}

	if priv {
		privServer := generateServer(privPort, appPrivPort, tlsConnPolicy, protocol)
		servers["privServer"] = privServer
	}

	return servers, nil
}

func generateServer(port int32, appPort int32, tlsConnPolicy []*caddytls.ConnectionPolicy, protocol string) *caddyhttp.Server {

	var warnings []caddyconfig.Warning

	reverseProxy := reverseproxy.Handler{
		Upstreams: []*reverseproxy.Upstream{{
			Dial: fmt.Sprintf("localhost:%d", appPort),
		}},
	}

	// Set transport protocol if specified
	if protocol != "" {
		transport := &reverseproxy.HTTPTransport{}
		if protocol == "h2c" {
			transport.Versions = []string{"h2c"}
		}
		reverseProxy.TransportRaw = caddyconfig.JSONModuleObject(transport, "protocol", protocol, &warnings)
	}

	server := &caddyhttp.Server{
		Listen: []string{fmt.Sprintf(":%d", port)},
		AutoHTTPS: &caddyhttp.AutoHTTPSConfig{
			Disabled: true,
		},
		Routes: caddyhttp.RouteList{{
			HandlersRaw: []json.RawMessage{
				caddyconfig.JSONModuleObject(reverseProxy, "handler", "reverse_proxy", &warnings),
			},
		}},
		TLSConnPolicies: tlsConnPolicy,
	}

	return server
}

func generateCaddyConfig(pub bool, priv bool, pubPort int32, privPort int32, pubH2C bool, privH2C bool, pubH2CPort int32, privH2CPort int32, env *crd.ClowdEnvironment) (string, error) {
	var warnings []caddyconfig.Warning

	var httpServers map[string]*caddyhttp.Server
	var h2cServers map[string]*caddyhttp.Server
	var err error

	appPubPort := env.Spec.Providers.Web.Port
	appPrivPort := env.Spec.Providers.Web.PrivatePort
	appH2CPubPort := env.Spec.Providers.Web.H2CPort
	appH2CPrivPort := env.Spec.Providers.Web.H2CPrivatePort

	// Generate HTTP servers
	httpServers, err = generateServers(pub, priv, pubPort, privPort, appPubPort, appPrivPort, "http")
	if err != nil {
		fmt.Print("error generating caddy HTTP server config. Server generation failed")
	}

	// Generate H2C servers
	h2cServers, err = generateServers(pubH2C, privH2C, pubH2CPort, privH2CPort, appH2CPubPort, appH2CPrivPort, "h2c")
	if err != nil {
		fmt.Print("error generating caddy H2C server config. Server generation failed")
	}

	fl := caddytls.FileLoader{{
		Certificate: "/certs/tls.crt",
		Key:         "/certs/tls.key",
		Tags:        []string{"cert0"},
	}}

	tlsConfig := caddytls.TLS{
		CertificatesRaw: caddy.ModuleMap{"load_files": caddyconfig.JSON(fl, &warnings)},
	}

	appsRaw := map[string]json.RawMessage{
		"tls": caddyconfig.JSON(tlsConfig, &warnings),
	}

	// Add HTTP app if there are HTTP servers
	if len(httpServers) > 0 {
		httpAppConfig := caddyhttp.App{
			Servers: httpServers,
		}
		appsRaw["http"] = caddyconfig.JSON(httpAppConfig, &warnings)
	}

	// Add H2C app if there are H2C servers
	if len(h2cServers) > 0 {
		h2cAppConfig := caddyhttp.App{
			Servers: h2cServers,
		}
		appsRaw["h2c"] = caddyconfig.JSON(h2cAppConfig, &warnings)
	}

	v := caddy.Config{
		StorageRaw: []byte{},
		AppsRaw:    appsRaw,
	}

	pretty, _ := json.MarshalIndent(v, "", "  ")
	return string(pretty), nil
}
