package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
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

	err := mp.configureKeycloak()

	if err != nil {
		newErr := errors.Wrap("couldn't config", err)
		newErr.Requeue = true
		return nil, newErr
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
		Image:          "quay.io/cloudservices/mbop:513688c",
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

type KeyCloakClient struct {
	BaseURL     string
	Username    string
	Password    string
	AccessToken string
	Ctx         context.Context
}

func NewKeyCloakClient(BaseUrl string, Username string, Password string, BaseCtx context.Context) (*KeyCloakClient, error) {
	client := KeyCloakClient{
		BaseURL:  BaseUrl,
		Username: Username,
		Password: Password,
		Ctx:      BaseCtx,
	}
	err := client.init()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (k *KeyCloakClient) init() error {

	headers := map[string]string{
		"Content-type": "application/x-www-form-urlencoded",
	}

	resp, err := k.rawMethod("POST", "/auth/realms/master/protocol/openid-connect/token", "grant_type=password&client_id=admin-cli&username=admin&password=admin", headers)
	if err != nil {
		return err
	}

	var iface interface{}

	err = json.NewDecoder(resp.Body).Decode(&iface)

	if err != nil {
		return err
	}

	auth, ok := iface.(map[string]interface{})

	if !ok {
		return fmt.Errorf("could not get auth info")
	}

	access_token, ok := auth["access_token"]

	if !ok {
		return fmt.Errorf("could not get access token")
	}

	access_token_string := access_token.(string)

	k.AccessToken = access_token_string

	return nil
}

func (k *KeyCloakClient) rawMethod(method string, url string, body string, headers map[string]string) (*http.Response, error) {
	fullUrl := fmt.Sprintf("%s%s", k.BaseURL, url)
	fmt.Printf("\n\n%v\n%v\n%v\n%v\n\n", url, body, headers, method)
	ctx, cancel := context.WithTimeout(k.Ctx, 10*time.Second)
	defer cancel()

	r := strings.NewReader(body)

	req, err := http.NewRequestWithContext(ctx, method, fullUrl, r)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (k *KeyCloakClient) Get(url string, body string, headers map[string]string) (*http.Response, error) {
	headers["Authorization"] = fmt.Sprintf("Bearer %s", k.AccessToken)
	return k.rawMethod("GET", url, body, headers)
}

func (k *KeyCloakClient) Post(url string, body string, headers map[string]string) (*http.Response, error) {
	headers["Authorization"] = fmt.Sprintf("Bearer %s", k.AccessToken)

	return k.rawMethod("POST", url, body, headers)
}

type Realm struct {
	Realm string `json:"realm"`
}

type RealmResponse []Realm

func (k *KeyCloakClient) doesRealmExist(requestedRealmName string) (bool, error) {
	resp, err := k.Get("/auth/admin/realms", "", make(map[string]string))

	if err != nil {
		return false, err
	}

	iface := &RealmResponse{}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(data, iface)

	if err != nil {
		return false, err
	}

	for _, realm := range *iface {
		if realm.Realm == requestedRealmName {
			return true, nil
		}
	}
	return false, nil
}

type Client struct {
	ClientId string `json:"clientId"`
}

type ClientResponse []Client

func (k *KeyCloakClient) doesClientExist(realm string, requestedClientName string) (bool, error) {
	resp, err := k.Get(fmt.Sprintf("/auth/admin/realms/%s/clients", realm), "", make(map[string]string))

	if err != nil {
		return false, err
	}

	iface := &ClientResponse{}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(data, iface)

	if err != nil {
		return false, err
	}

	for _, client := range *iface {
		fmt.Printf("\n\n%s\n\n", client.ClientId)
		if client.ClientId == requestedClientName {
			return true, nil
		}
	}
	return false, nil
}

type createRealmStruct struct {
	Realm   string `json:"realm"`
	Enabled bool   `json:"enabled"`
	ID      string `json:"id"`
}

func (k *KeyCloakClient) createRealm(requestedRealmName string) error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	postObj := createRealmStruct{
		Realm:   requestedRealmName,
		Enabled: true,
		ID:      requestedRealmName,
	}

	b, err := json.Marshal(postObj)

	if err != nil {
		return err
	}

	resp, err := k.Post("/auth/admin/realms", string(b), headers)

	if err != nil {
		return err
	}

	fmt.Printf("\n\n%v\n\n", resp)
	v, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("\n\n%s\n\n", string(v))
	return nil
}

type mapperConfig struct {
	UserInfoTokenClaim bool   `json:"userinfo.token.claim"`
	UserAttribute      string `json:"user.Attribute"`
	IDTokenClaim       bool   `json:"id.token.claim"`
	AccessTokenClaim   bool   `json:"access.token.claim"`
	ClaimName          string `json:"claim.name"`
	JSONTypeLabel      string `json:"jsonType.label"`
}

type mapperStruct struct {
	Name            string       `json:"name"`
	ID              string       `json:"id"`
	Protocol        string       `json:"protocol"`
	ProtocolMapper  string       `json:"protocolMapper"`
	ConsentRequired bool         `json:"consentRequired"`
	Config          mapperConfig `json:"config"`
}

func createMapper(attr string, mtype string) mapperStruct {
	return mapperStruct{
		Name:            attr,
		ID:              attr,
		Protocol:        "openid-connect",
		ProtocolMapper:  "oidc-usermodel-attribute-mapper",
		ConsentRequired: false,
		Config: mapperConfig{
			UserInfoTokenClaim: true,
			UserAttribute:      attr,
			IDTokenClaim:       true,
			AccessTokenClaim:   true,
			ClaimName:          attr,
			JSONTypeLabel:      mtype,
		},
	}
}

type clientStruct struct {
	ClientId                  string         `json:"clientId"`
	Enabled                   bool           `json:"enabled"`
	BearerOnly                bool           `json:"bearerOnly"`
	PublicClient              bool           `json:"publicClient"`
	RedirectUris              []string       `json:"redirectUris"`
	WebOrigins                []string       `json:"webOrigins"`
	ProtocolMappers           []mapperStruct `json:"protocolMappers"`
	DirectAccessGrantsEnabled bool           `json:"directAccessGrantsEnabled"`
}

func (k *KeyCloakClient) createClient(realmName string, clientName string) error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	postObj := clientStruct{
		ClientId:                  clientName,
		Enabled:                   true,
		BearerOnly:                false,
		PublicClient:              true,
		RedirectUris:              []string{"url"},
		WebOrigins:                []string{"url"},
		DirectAccessGrantsEnabled: true,
		ProtocolMappers: []mapperStruct{
			createMapper("account_number", "String"),
			createMapper("account_number", "String"),
			createMapper("account_id", "String"),
			createMapper("org_id", "String"),
			createMapper("username", "String"),
			createMapper("email", "String"),
			createMapper("first_name", "String"),
			createMapper("last_name", "String"),
			createMapper("is_org_admin", "boolean"),
			createMapper("is_internal", "boolean"),
			createMapper("is_active", "boolean"),
		},
	}

	b, err := json.Marshal(postObj)

	if err != nil {
		return err
	}

	resp, err := k.Post(
		fmt.Sprintf("/auth/admin/realms/%s/clients", realmName),
		string(b), headers,
	)

	if err != nil {
		return err
	}

	fmt.Printf("\n\n%v\n\n", resp)
	v, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("\n\n%s\n\n", string(v))
	return nil
}

func (m *mockProvider) configureKeycloak() error {
	s := &core.Service{}
	if err := m.Cache.Get(MockKeycloakService, s); err != nil {
		return err
	}

	client, err := NewKeyCloakClient(fmt.Sprintf("http://%s.%s.svc:8080", s.Name, s.Namespace), "admin", "admin", m.Ctx)

	if err != nil {
		return err
	}

	exists, err := client.doesRealmExist("redhat-external")

	if err != nil {
		return err
	}

	if !exists {
		err := client.createRealm("redhat-external")
		if err != nil {
			return err
		}
	}

	exists, err = client.doesClientExist("redhat-external", "cloud-services")

	if err != nil {
		return err
	}

	if !exists {
		err := client.createClient("redhat-external", "cloud-services")
		if err != nil {
			return err
		}
	}

	fmt.Printf("\n\n%v\n\n", exists)

	return nil
}
