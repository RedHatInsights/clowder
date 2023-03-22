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
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func generateTLSContext() (*anypb.Any, error) {
	return anypb.New(&tls.DownstreamTlsContext{
		CommonTlsContext: &tls.CommonTlsContext{
			TlsCertificates: []*tls.TlsCertificate{{
				CertificateChain: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "/certs/tls.crt",
					},
				},
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_Filename{
						Filename: "/certs/tls.key",
					},
				},
			}},
		},
	})
}

func generateHTTPConnectionManager(cluster string) (*anypb.Any, error) {
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
									Cluster: cluster,
								},
								Timeout: &durationpb.Duration{
									Seconds: 600,
								},
							},
						},
					}},
				}},
			},
		},
	})
}

func generateListener(cluster string, port uint32, name string) (*listener.Listener, error) {
	tlsContextObj, err := generateTLSContext()
	if err != nil {
		return nil, err
	}

	httpConectionManagerObj, err := generateHTTPConnectionManager(cluster)
	if err != nil {
		return nil, err
	}

	return &listener.Listener{
		Name: name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
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
	}, nil
}

func generateListeners(pub bool, priv bool, pubPort uint32, privPort uint32) ([]*listener.Listener, error) {
	listeners := []*listener.Listener{}

	if pub {
		listener, err := generateListener("public_endpoint", pubPort, "public")
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
	}
	if priv {
		listener, err := generateListener("private_endpoint", privPort, "private")
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

func generateCluster(name string, port uint32) *cluster.Cluster {
	return &cluster.Cluster{
		Name: name, LoadAssignment: &endpoint.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*endpoint.LbEndpoint{{
					HostIdentifier: &endpoint.LbEndpoint_Endpoint{
						Endpoint: &endpoint.Endpoint{
							Address: &core.Address{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Address: "127.0.0.1",
										PortSpecifier: &core.SocketAddress_PortValue{
											PortValue: port,
										},
									},
								},
							},
						},
					},
				}},
			}},
		},
	}
}

func generateClusters(pub bool, priv bool) []*cluster.Cluster {
	clusters := []*cluster.Cluster{}
	if pub {
		clusters = append(clusters, generateCluster("public_endpoint", 8000))
	}
	if priv {
		clusters = append(clusters, generateCluster("private_endpoint", 10000))
	}
	return clusters
}

func generateEnvoyConfig(pub bool, priv bool, pubPort uint32, privPort uint32) (string, error) {

	beat := &envoy.Bootstrap{}
	beat.StaticResources = &envoy.Bootstrap_StaticResources{}

	listeners, err := generateListeners(pub, priv, pubPort, privPort)
	if err != nil {
		return "", err
	}

	beat.StaticResources.Listeners = listeners

	clusters := generateClusters(pub, priv)

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

func addCertVolume(d *apps.Deployment, dnn string) {
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, v1.Volume{
		Name: "tls-ca",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "openshift-service-ca.crt",
				},
			},
		},
	})
	for i, container := range d.Spec.Template.Spec.Containers {
		vms := container.VolumeMounts
		if container.Name == dnn {
			vms = append(vms, v1.VolumeMount{
				Name:      "tls-ca",
				ReadOnly:  true,
				MountPath: "/cdapp/certs",
			})
		}
		d.Spec.Template.Spec.Containers[i].VolumeMounts = vms
	}

	for i, iContainer := range d.Spec.Template.Spec.InitContainers {
		vms := iContainer.VolumeMounts
		vms = append(vms, v1.VolumeMount{
			Name:      "tls-ca",
			ReadOnly:  true,
			MountPath: "/cdapp/certs",
		})
		d.Spec.Template.Spec.InitContainers[i].VolumeMounts = vms
	}
}
