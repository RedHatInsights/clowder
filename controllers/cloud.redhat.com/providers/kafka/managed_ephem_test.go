package kafka

import (
	"context"
	"io"
	"net/http"
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/clientcredentials"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type MockHTTPClient struct {
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return nil, nil
}
func (m *MockHTTPClient) Get(url string) (resp *http.Response, err error) {
	return nil, nil
}
func (m *MockHTTPClient) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	return nil, nil
}

func init() {
	ClientCreator = func(provider *providers.Provider, clientCred clientcredentials.Config) HTTPClient {
		return &MockHTTPClient{}
	}
}

func makeJSON(jsonToParse string) *apiextensions.JSON {
	var parsedJSON apiextensions.JSON
	parsedJSON.UnmarshalJSON([]byte(jsonToParse))
	return &parsedJSON
}

func makeMockProvider() providers.Provider {
	m := providers.Provider{}
	limit := makeJSON(`{
        "cpu": "600m",
        "memory": "800Mi"
	}`)
	env := crd.ClowdEnvironment{}
	m.Env = &env
	m.Env.Name = "test"
	m.Env.Spec.Providers.Kafka.Connect.Resources.Limits = limit
	req := makeJSON(`{
        "cpu": "300m",
        "memory": "500Mi"
	}`)
	kafkaConfig := crd.KafkaConfig{}
	m.Env.Spec.Providers.Kafka = kafkaConfig
	m.Ctx = context.Background()
	m.Env.Spec.Providers.Kafka.Connect.Resources.Requests = req
	m.Env.Spec.Providers.Kafka.Connect.Replicas = 4
	m.Env.Spec.Providers.Kafka.Connect.Version = "3.0.0"
	m.Env.Spec.Providers.Kafka.Connect.Image = IMAGE_KAFKA_XJOIN
	m.Env.Spec.Providers.Kafka.EnableLegacyStrimzi = false
	m.Cache = &resource_cache.ObjectCache{}
	return m
}

func makeMockSecretData() map[string][]byte {
	secretData := make(map[string][]byte)
	secretData["client.secret"] = []byte("Shh, tell no one.")
	return secretData
}

func TestHTTPClientCacheSet(t *testing.T) {
	hcc := newHTTPClientCahce()
	client := http.Client{}
	hcc.Set("test", &client)
}

func TestHTTPClientCacheGet(t *testing.T) {
	hcc := newHTTPClientCahce()
	client := http.Client{}
	hcc.Set("test", &client)
	_, ok := hcc.Get("test")
	assert.True(t, ok)
	_, notOK := hcc.Get("not-found")
	assert.False(t, notOK)
}

func TestHTTPClientCacheRemove(t *testing.T) {
	hcc := newHTTPClientCahce()
	client := http.Client{}
	hcc.Set("test", &client)
	hcc.Remove("test")
}

func TestKafkaConnectBuilderCreate(t *testing.T) {
	provider := makeMockProvider()
	assert.NotNil(t, provider)
	secretData := makeMockSecretData()
	builder := newKafkaConnectBuilder(provider, secretData)
	err := builder.Create()
	assert.NotNil(t, err)
	builder.BuildSpec()

}
