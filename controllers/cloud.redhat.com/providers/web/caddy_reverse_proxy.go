package web

import (
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	caddy "github.com/caddyserver/caddy/v2"
	caddyconfig "github.com/caddyserver/caddy/v2/caddyconfig"
	caddyhttp "github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	caddytls "github.com/caddyserver/caddy/v2/modules/caddytls"
)

func generateServers(pub bool, priv bool, pubPort int32, privPort int32, appPubPort int32, appPrivPort int32) (map[string]*caddyhttp.Server, error) {
	servers := make(map[string]*caddyhttp.Server)

	tlsConnPolicy := []*caddytls.ConnectionPolicy{{
		CertSelection: &caddytls.CustomCertSelectionPolicy{
			AnyTag: []string{"cert0"},
		},
	}}

	if pub {
		pubServer := generateServer(pubPort, appPubPort, tlsConnPolicy)
		servers["pubServer"] = pubServer
	}

	if priv {
		privServer := generateServer(privPort, appPrivPort, tlsConnPolicy)
		servers["privServer"] = privServer
	}

	return servers, nil
}

func generateServer(port int32, appPort int32, tlsConnPolicy []*caddytls.ConnectionPolicy) *caddyhttp.Server {

	var warnings []caddyconfig.Warning

	reverseProxy := reverseproxy.Handler{
		Upstreams: []*reverseproxy.Upstream{{
			Dial: fmt.Sprintf("localhost:%d", appPort),
		}},
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

func generateCaddyConfig(pub bool, priv bool, pubPort int32, privPort int32, env *crd.ClowdEnvironment) (string, error) {
	var warnings []caddyconfig.Warning

	var servers map[string]*caddyhttp.Server
	var err error

	appPubPort := env.Spec.Providers.Web.Port
	appPrivPort := env.Spec.Providers.Web.PrivatePort

	servers, err = generateServers(pub, priv, pubPort, privPort, appPubPort, appPrivPort)
	if err != nil {
		fmt.Print("error generating caddy server config. Server generation failed")
	}

	appConfig := caddyhttp.App{
		Servers: servers,
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
