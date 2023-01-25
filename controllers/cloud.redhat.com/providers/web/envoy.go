package web

import (
	envoy "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	httpconman "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

func generateTLSContext() (*anypb.Any, error) {
	return anypb.New(&tls.DownstreamTlsContext{
		CommonTlsContext: &tls.CommonTlsContext{
			TlsCertificates: []*tls.TlsCertificate{{
				CertificateChain: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "/certs/cert.pem",
					},
				},
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "/certs/key.pem",
					},
				},
			}},
		},
	})
}

func generateHTTPConnectionManager() (*anypb.Any, error) {

	routerObj, err := anypb.New(&router.Router{})
	if err != nil {
		return nil, err
	}

	return anypb.New(&httpconman.HttpConnectionManager{
		StatPrefix: "ingress_http",
		HttpFilters: []*httpconman.HttpFilter{{
			Name: "envoy.filters.http.router",
			ConfigType: &httpconman.HttpFilter_TypedConfig{
				TypedConfig: routerObj,
			},
		}},
		RouteSpecifier: &httpconman.HttpConnectionManager_RouteConfig{
			RouteConfig: &route.RouteConfiguration{
				VirtualHosts: []*route.VirtualHost{{
					Name:    "local_service",
					Domains: []string{"*"},
					Routes: []*route.Route{{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_Cluster{
									Cluster: "service_envoyproxy_io",
								},
							},
						},
					}},
				}},
			},
		},
	})
}

func generateListeners() ([]*listener.Listener, error) {
	tlsContextObj, err := generateTLSContext()
	if err != nil {
		return nil, err
	}

	httpConectionManagerObj, err := generateHTTPConnectionManager()
	if err != nil {
		return nil, err
	}

	return []*listener.Listener{{
		Name: "listener_0",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 8800,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: "envoy.filters.network.http_connection_manager",
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: httpConectionManagerObj,
				},
			}},
			TransportSocket: &core.TransportSocket{
				Name: "envoy.transport_sockets.tls",
				ConfigType: &core.TransportSocket_TypedConfig{
					TypedConfig: tlsContextObj,
				},
			},
		}},
	}}, nil
}

func generateClusters() ([]*cluster.Cluster, error) {
	return []*cluster.Cluster{{
		Name: "service_envoyproxy_io", LoadAssignment: &endpoint.ClusterLoadAssignment{
			ClusterName: "service_envoyproxy_io",
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*endpoint.LbEndpoint{{
					HostIdentifier: &endpoint.LbEndpoint_Endpoint{
						Endpoint: &endpoint.Endpoint{
							Address: &core.Address{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Address: "127.0.0.1",
										PortSpecifier: &core.SocketAddress_PortValue{
											PortValue: 8000,
										},
									},
								},
							},
						},
					},
				}},
			}},
		},
	}}, nil
}

func generateEnvoyConfig() (string, error) {

	beat := &envoy.Bootstrap{}
	beat.StaticResources = &envoy.Bootstrap_StaticResources{}

	listeners, err := generateListeners()
	if err != nil {
		return "", err
	}

	beat.StaticResources.Listeners = listeners

	clusters, err := generateClusters()
	if err != nil {
		return "", err
	}

	beat.StaticResources.Clusters = clusters
	err = beat.Validate()
	if err != nil {
		return "", err
	}

	configString, err := protojson.Marshal(beat)
	if err != nil {
		return "", err
	}
	return string(configString), nil
}
