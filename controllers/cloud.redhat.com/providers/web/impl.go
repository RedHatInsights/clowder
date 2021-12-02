package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	obj "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
)

func makeService(cache *rc.ObjectCache, deployment *crd.Deployment, app *crd.ClowdApp, env *crd.ClowdEnvironment) error {

	s := &core.Service{}
	nn := app.GetDeploymentNamespacedName(deployment)

	if err := cache.Create(CoreService, nn, s); err != nil {
		return err
	}

	d := &apps.Deployment{}

	cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(deployment))

	servicePorts := []core.ServicePort{}
	containerPorts := []core.ContainerPort{}

	if bool(deployment.Web) || deployment.WebServices.Public.Enabled {
		appProtocol := "http"
		// Create the core service port
		webPort := core.ServicePort{
			Name:        "public",
			Port:        env.Spec.Providers.Web.Port,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
		}

		servicePorts = append(servicePorts, webPort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "web",
				ContainerPort: env.Spec.Providers.Web.Port,
			},
		)

		if env.Spec.Providers.Web.Mode == "local" {
			authPortNumber := env.Spec.Providers.Web.AuthPort

			if authPortNumber == 0 {
				authPortNumber = 8080
			}
			authPort := core.ServicePort{
				Name:        "auth",
				Port:        authPortNumber,
				Protocol:    "TCP",
				AppProtocol: &appProtocol,
			}
			servicePorts = append(servicePorts, authPort)
		}
	}

	if deployment.WebServices.Private.Enabled {
		privatePort := env.Spec.Providers.Web.PrivatePort

		appProtocol := "http"
		if deployment.WebServices.Private.AppProtocol != "" {
			appProtocol = string(deployment.WebServices.Private.AppProtocol)
		}

		if privatePort == 0 {
			privatePort = 10000
		}

		webPort := core.ServicePort{
			Name:        "private",
			Port:        privatePort,
			Protocol:    "TCP",
			AppProtocol: &appProtocol,
		}
		servicePorts = append(servicePorts, webPort)

		// Append port to deployment spec
		containerPorts = append(containerPorts,
			core.ContainerPort{
				Name:          "private",
				ContainerPort: privatePort,
			},
		)
	}

	utils.MakeService(s, nn, map[string]string{"pod": nn.Name}, servicePorts, app, env.IsNodePort())

	d.Spec.Template.Spec.Containers[0].Ports = containerPorts

	if err := cache.Update(CoreService, s); err != nil {
		return err
	}

	if err := cache.Update(deployProvider.CoreDeployment, d); err != nil {
		return err
	}

	return nil
}

func makeKeycloak(o obj.ClowdObject, objMap providers.ObjectMap, usePVC bool, nodePort bool) {
	nn := providers.GetNamespacedName(o, "keycloak")

	dd := objMap[WebKeycloakDeployment].(*apps.Deployment)
	svc := objMap[WebKeycloakService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

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
			Name: "KEYCLOAK_USER",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: nn.Name,
					},
					Key: "username",
				},
			},
		},
		{
			Name: "KEYCLOAK_PASSWORD",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: nn.Name,
					},
					Key: "password",
				},
			},
		},
	}

	port := int32(8080)

	ports := []core.ContainerPort{{
		Name:          "service",
		ContainerPort: port,
	}}

	probeHandler := core.Handler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port,
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

	image := "quay.io/keycloak/keycloak:11.0.3"

	if clowderconfig.LoadedConfig.Images.Keycloak != "" {
		image = clowderconfig.LoadedConfig.Images.Keycloak
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          image,
		Env:            envVars,
		Ports:          ports,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		Resources: core.ResourceRequirements{
			Limits: core.ResourceList{
				"memory": resource.MustParse("750Mi"),
				"cpu":    resource.MustParse("1"),
			},
			Requests: core.ResourceList{
				"memory": resource.MustParse("400Mi"),
				"cpu":    resource.MustParse("100m"),
			},
		},
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
	snn := providers.GetNamespacedName(o, "keycloak")
	nn := providers.GetNamespacedName(o, "mbop")

	dd := objMap[WebBOPDeployment].(*apps.Deployment)
	svc := objMap[WebBOPService].(*core.Service)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name

	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	replicas := int32(1)

	dd.Spec.Replicas = &replicas
	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	dd.Spec.Template.ObjectMeta.Labels = labels

	envVars := []core.EnvVar{
		{
			Name:  "ALL_PASS",
			Value: "true",
		},
		{
			Name:  "KEYCLOAK_SERVER",
			Value: fmt.Sprintf("http://%s-keycloak.%s.svc:8080", o.GetClowdName(), o.GetClowdNamespace()),
		},
		{
			Name: "KEYCLOAK_USERNAME",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "username",
				},
			},
		},
		{
			Name: "KEYCLOAK_PASSWORD",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{
						Name: snn.Name,
					},
					Key: "password",
				},
			},
		},
	}

	port := int32(8080)

	ports := []core.ContainerPort{{
		Name:          "service",
		ContainerPort: port,
	}}

	probeHandler := core.Handler{
		TCPSocket: &core.TCPSocketAction{
			Port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port,
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

	image := "quay.io/cloudservices/mbop:a8be5e9"

	if clowderconfig.LoadedConfig.Images.MBOP != "" {
		image = clowderconfig.LoadedConfig.Images.MBOP
	}

	c := core.Container{
		Name:           nn.Name,
		Image:          image,
		Env:            envVars,
		Ports:          ports,
		LivenessProbe:  &livenessProbe,
		ReadinessProbe: &readinessProbe,
		Resources: core.ResourceRequirements{
			Limits: core.ResourceList{
				"memory": resource.MustParse("500Mi"),
				"cpu":    resource.MustParse("1"),
			},
			Requests: core.ResourceList{
				"memory": resource.MustParse("384Mi"),
				"cpu":    resource.MustParse("100m"),
			},
		},
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
	Log         logr.Logger
}

func NewKeyCloakClient(BaseUrl string, Username string, Password string, BaseCtx context.Context, Log logr.Logger) (*KeyCloakClient, error) {
	log := Log.WithValues("subsystem", "KeyCloakClient")
	client := KeyCloakClient{
		BaseURL:  BaseUrl,
		Username: Username,
		Password: Password,
		Ctx:      BaseCtx,
		Log:      log,
	}
	err := client.init()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

type AuthStruct struct {
	AccessToken string `json:"access_token"`
}

func (k *KeyCloakClient) init() error {

	headers := map[string]string{
		"Content-type": "application/x-www-form-urlencoded",
	}

	resp, err := k.rawMethod(
		"POST",
		"/auth/realms/master/protocol/openid-connect/token",
		fmt.Sprintf(
			"grant_type=password&client_id=admin-cli&username=%s&password=%s",
			k.Username,
			k.Password,
		),
		headers,
	)
	if err != nil {
		return err
	}

	respObj := &AuthStruct{}

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	json.Unmarshal(data, respObj)

	k.AccessToken = respObj.AccessToken

	return nil
}

func (k *KeyCloakClient) rawMethod(method string, url string, body string, headers map[string]string) (*http.Response, error) {
	fullUrl := fmt.Sprintf("%s%s", k.BaseURL, url)

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

	k.Log.Info(fmt.Sprintf("%s - %s - %d", url, method, resp.StatusCode))

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
		if client.ClientId == requestedClientName {
			return true, nil
		}
	}
	return false, nil
}

type User struct {
	Username string `json:"username"`
}

type UserResponse []User

func (k *KeyCloakClient) doesUserExist(realm string, requestedUsername string) (bool, error) {
	resp, err := k.Get(fmt.Sprintf("/auth/admin/realms/%s/users", realm), "", make(map[string]string))

	if err != nil {
		return false, err
	}

	iface := &UserResponse{}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(data, iface)

	if err != nil {
		return false, err
	}

	for _, user := range *iface {
		if user.Username == requestedUsername {
			return true, nil
		}
	}
	return false, nil
}

type createUserStruct struct {
	Enabled     bool              `json:"enabled"`
	Username    string            `json:"username"`
	FirstName   string            `json:"firstName"`
	LastName    string            `json:"lastName"`
	Email       string            `json:"email"`
	Attributes  userAttributes    `json:"attributes"`
	Credentials []userCredentials `json:"credentials"`
}

type userAttributes struct {
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	AccountID     string `json:"account_id"`
	AccountNumber string `json:"account_number"`
	OrdID         string `json:"org_id"`
	IsInternal    bool   `json:"is_internal"`
	IsOrgAdmin    bool   `json:"is_org_admin"`
	IsActive      bool   `json:"is_active"`
	Entitlements  string `json:"entitlements"`
}

type userCredentials struct {
	Temporary bool   `json:"temporary"`
	Type      string `json:"type"`
	Value     string `json:"value"`
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
		v, _ := ioutil.ReadAll(resp.Body)
		k.Log.Error(err, string(v))
		return err
	}

	return nil
}

type mapperConfig struct {
	UserInfoTokenClaim bool   `json:"userinfo.token.claim"`
	UserAttribute      string `json:"user.attribute"`
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
	BaseURL                   string         `json:"baseUrl"`
	RedirectUris              []string       `json:"redirectUris"`
	WebOrigins                []string       `json:"webOrigins"`
	ProtocolMappers           []mapperStruct `json:"protocolMappers"`
	DirectAccessGrantsEnabled bool           `json:"directAccessGrantsEnabled"`
}

func (k *KeyCloakClient) createClient(realmName, clientName, envName string) error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	postObj := clientStruct{
		ClientId:                  clientName,
		Enabled:                   true,
		BearerOnly:                false,
		PublicClient:              true,
		RedirectUris:              []string{"*"},
		WebOrigins:                []string{"*"},
		BaseURL:                   fmt.Sprintf("https://%s", envName),
		DirectAccessGrantsEnabled: true,
		ProtocolMappers: []mapperStruct{
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
			createMapper("entitlements", "String"),
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
		v, _ := ioutil.ReadAll(resp.Body)
		k.Log.Error(err, string(v))
		return err
	}

	return nil
}

func (k *KeyCloakClient) createUser(realmName string, user *createUserStruct) error {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	b, err := json.Marshal(user)

	if err != nil {
		return err
	}

	resp, err := k.Post(
		fmt.Sprintf("/auth/admin/realms/%s/users", realmName),
		string(b), headers,
	)

	if err != nil {
		v, _ := ioutil.ReadAll(resp.Body)
		k.Log.Error(err, string(v))
		return err
	}

	return nil
}

func (m *localWebProvider) configureKeycloak() error {
	s := &core.Service{}
	if err := m.Cache.Get(WebKeycloakService, s); err != nil {
		return err
	}

	client, err := NewKeyCloakClient(fmt.Sprintf("http://%s.%s.svc:8080", s.Name, s.Namespace), m.config.KeycloakConfig.Username, m.config.KeycloakConfig.Password, m.Ctx, m.Log)

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
		err := client.createClient("redhat-external", "cloud-services", m.Env.Name)
		if err != nil {
			return err
		}
	}

	exists, err = client.doesUserExist("redhat-external", m.config.KeycloakConfig.DefaultUsername)

	if err != nil {
		return err
	}

	m.Log.Info(fmt.Sprintf("User exists: %s", strconv.FormatBool(exists)))

	if !exists {

		user := &createUserStruct{
			Enabled:   true,
			Username:  "jdoe",
			FirstName: "John",
			LastName:  "Doe",
			Email:     "jdoe@example.com",
			Attributes: userAttributes{
				FirstName:     "John",
				LastName:      "Doe",
				AccountID:     "12345",
				AccountNumber: "12345",
				OrdID:         "12345",
				IsInternal:    false,
				IsOrgAdmin:    true,
				IsActive:      true,
				Entitlements:  `{"insights": {"is_entitled": true, "is_trial": false}}`,
			},
			Credentials: []userCredentials{{
				Temporary: false,
				Type:      "password",
				Value:     m.config.KeycloakConfig.DefaultPassword,
			}},
		}

		err := client.createUser("redhat-external", user)
		if err != nil {
			return err
		}
	}

	return nil
}
