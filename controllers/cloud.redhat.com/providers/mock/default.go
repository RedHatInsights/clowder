package mock

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type mockConfig struct {
	BOPURL      string
	KeycloakURL string
}

type mockProvider struct {
	providers.Provider
	config mockConfig
}

// NewMockProvider returns a new mock provider
func NewMockProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	if !p.Env.Spec.Providers.Mock {
		return &mockProvider{Provider: *p}, nil
	}
	mp := &mockProvider{Provider: *p}

	objList := []providers.ResourceIdent{
		MockKeycloakDeployment,
		MockKeycloakService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "keycloak", makeKeycloak, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	objList = []providers.ResourceIdent{
		MockBOPDeployment,
		MockBOPService,
	}

	if err := providers.CachedMakeComponent(p.Cache, objList, p.Env, "mbop", makeBOP, false, p.Env.IsNodePort()); err != nil {
		return nil, err
	}

	mp.config.BOPURL = fmt.Sprintf("http://%s-%s.%s.svc:8080", p.Env.GetClowdName(), "mbop", p.Env.GetClowdNamespace())
	mp.config.KeycloakURL = fmt.Sprintf("http://%s-%s.%s.svc:8080", p.Env.GetClowdName(), "keycloak", p.Env.GetClowdNamespace())

	nn := types.NamespacedName{
		Name:      "caddy-config",
		Namespace: p.Env.GetClowdNamespace(),
	}

	sec := &core.Secret{}
	if err := p.Cache.Create(MockSecret, nn, sec); err != nil {
		return nil, err
	}

	sec.Name = nn.Name
	sec.Namespace = nn.Namespace
	sec.ObjectMeta.OwnerReferences = []metav1.OwnerReference{p.Env.MakeOwnerReference()}
	sec.Type = core.SecretTypeOpaque

	sec.StringData = map[string]string{
		"bopurl":      mp.config.BOPURL,
		"keycloakurl": mp.config.KeycloakURL,
	}

	if err := p.Cache.Update(MockSecret, sec); err != nil {
		return nil, err
	}

	return mp, nil
}

func (m *mockProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	c.Mock = &config.MockConfig{
		Bop:      providers.StrPtr(m.config.BOPURL),
		Keycloak: providers.StrPtr(m.config.KeycloakURL),
	}
	return nil
}

func makeKeycloak(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
	nn := providers.GetNamespacedName(o, "keycloak")

	dd := objMap[MockKeycloakDeployment].(*apps.Deployment)
	svc := objMap[MockKeycloakService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{{
		Name: "quay-cloudservices-pull",
	}}

	// get the secret

	port := int32(8080)

	envVars := []core.EnvVar{
		{
			Name:  "DB_VENDOR",
			Value: "h2",
		},
		{
			Name:  "PROXY_ADDRESS_FORWARDING",
			Value: "true",
		},
		{
			Name:  "KEYCLOAK_USER",
			Value: "admin",
		},
		{
			Name:  "KEYCLOAK_PASSWORD",
			Value: "admin",
		},
	}

	ports := []core.ContainerPort{{
		Name:          "service",
		ContainerPort: port,
	}}

	probeHandler := core.Handler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 8080,
			},
		},
	}

	livenessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 10,
		TimeoutSeconds:      2,
	}
	readinessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      2,
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          "quay.io/keycloak/keycloak:11.0.3",
		Env:            envVars,
		Ports:          ports,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:     "keycloak",
		Port:     port,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)

}

func makeBOP(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
	nn := providers.GetNamespacedName(o, "mbop")

	dd := objMap[MockBOPDeployment].(*apps.Deployment)
	svc := objMap[MockBOPService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	dd.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{{
		Name: "quay-cloudservices-pull",
	}}

	// get the secret

	port := int32(8080)

	envVars := []core.EnvVar{
		{
			Name:  "KEYCLOAK_SERVER",
			Value: fmt.Sprintf("http://%s-keycloak.%s.svc:8080", o.GetClowdName(), o.GetClowdNamespace()),
		},
	}

	ports := []core.ContainerPort{{
		Name:          "service",
		ContainerPort: port,
	}}

	probeHandler := core.Handler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 8080,
			},
		},
	}

	livenessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 10,
		TimeoutSeconds:      2,
	}
	readinessProbe := core.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 20,
		TimeoutSeconds:      2,
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          "127.0.0.1:5000/mbop:4",
		Env:            envVars,
		Ports:          ports,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
	}

	dd.Spec.Template.Spec.Containers = []core.Container{c}
	dd.Spec.Template.SetLabels(labels)

	servicePorts := []core.ServicePort{{
		Name:     "mbop",
		Port:     port,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o, nodePort)

}
