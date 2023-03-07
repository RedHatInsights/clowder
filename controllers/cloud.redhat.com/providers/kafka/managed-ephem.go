package kafka

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	core "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// KafkaConnect is the resource ident for a KafkaConnect object.
var EphemKafkaConnect = rc.NewSingleResourceIdent(ProvName, "kafka_connect", &strimzi.KafkaConnect{}, rc.ResourceOptions{WriteNow: true})

var EphemKafkaConnectSecret = rc.NewSingleResourceIdent(ProvName, "kafka_connect_secret", &core.Secret{}, rc.ResourceOptions{WriteNow: true})

type managedEphemProvider struct {
	providers.Provider
	secretData map[string][]byte
}

func NewManagedEphemKafka(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		EphemKafkaConnect,
		EphemKafkaConnectSecret,
		CyndiPipeline,
		CyndiAppSecret,
		CyndiHostInventoryAppSecret,
		CyndiConfigMap,
	)
	return &managedEphemProvider{Provider: *p}, nil
}

func (mep *managedEphemProvider) EnvProvide() error {
	sec, err := getSecret(&mep.Provider)
	if err != nil {
		return err
	}
	mep.secretData = sec.Data

	return mep.configureBrokers()
}

func (mep *managedEphemProvider) Provide(app *crd.ClowdApp) error {
	sec, err := getSecret(&mep.Provider)
	if err != nil {
		return err
	}

	username, password, hostname, adminHostname, tokenURL, cacert := destructureSecret(sec)

	httpClient := upsertClientCache(username, password, tokenURL, adminHostname, &mep.Provider)

	saslType := config.BrokerConfigAuthtypeSasl

	kconfig := config.KafkaConfig{
		Brokers: []config.BrokerConfig{{
			Hostname: hostname,
			Port:     utils.IntPtr(443),
			Authtype: &saslType,
			Sasl: &config.KafkaSASLConfig{
				Password:         &password,
				Username:         &username,
				SecurityProtocol: utils.StringPtr("SASL_SSL"),
				SaslMechanism:    utils.StringPtr("PLAIN"),
			},
			SecurityProtocol: utils.StringPtr("SASL_SSL"),
		}},
		Topics: []config.TopicConfig{},
	}

	if cacert != "" {
		kconfig.Brokers[0].Cacert = &cacert
	}

	mep.Config.Kafka = &kconfig

	if err := mep.configCyndi(app); err != nil {
		return err
	}

	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	if err := mep.processTopics(app, httpClient, adminHostname); err != nil {
		return err
	}

	return nil
}

func NewManagedEphemKafkaFinalizer(p *providers.Provider) error {

	if clowderconfig.LoadedConfig.Settings.ManagedKafkaEphemDeleteRegex == "" {
		return nil
	}

	sec, err := getSecret(p)
	if err != nil {
		return err
	}

	username, password, _, adminHostname, tokenURL, _ := destructureSecret(sec)

	rClient := upsertClientCache(username, password, tokenURL, adminHostname, p)

	topicList, err := getTopicList(rClient, adminHostname, p)
	if err != nil {
		return err
	}

	err = deleteTopics(topicList, rClient, adminHostname, p)

	return err
}

var (
	HTTP HTTPClient
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (resp *http.Response, err error)
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
}

var ClientCreator func(provider *providers.Provider, clientCred clientcredentials.Config) HTTPClient

func init() {
	ClientCreator = func(provider *providers.Provider, clientCred clientcredentials.Config) HTTPClient {
		client := clientCred.Client(provider.Ctx)
		return client
	}
}

type JSONPayload struct {
	Name     string   `json:"name"`
	Settings Settings `json:"settings"`
}

type Settings struct {
	NumPartitions int      `json:"numPartitions"`
	NumReplicas   int      `json:"numReplicas"`
	Config        []Config `json:"config"`
}

type Config struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type TopicsList struct {
	Items []Topic `json:"items"`
}

type Topic struct {
	Name string `json:"name"`
}

// Mutex protected cache of HTTP clients
var ClientCache = newHTTPClientCahce()

const ReplicaNumFloor = 3
const ReplicaNumCeiling = 0
const PartitionNumFloor = 3
const PartitionNumCeiling = 3

func deleteTopics(topicList *TopicsList, rClient HTTPClient, adminHostname string, p *providers.Provider) error {
	// "(env-)?ephemeral-.*"

	reg, err := regexp.Compile(fmt.Sprintf(".*%s.*", strings.ReplaceAll(p.Env.Name, "-", "[-\\.]")))
	if err != nil {
		return err
	}

	var regProtect *regexp.Regexp

	regProtect, err = regexp.Compile(clowderconfig.LoadedConfig.Settings.ManagedKafkaEphemDeleteRegex)
	if err != nil {
		return err
	}

	for _, topic := range topicList.Items {
		// The name of the environment must be in the topic names
		fmt.Print(reg.String())
		if reg.Find([]byte(topic.Name)) == nil {
			continue
		}

		// The name must also match the global topic protector
		if regProtect == nil || (regProtect != nil && regProtect.Find([]byte(topic.Name)) == nil) {
			continue
		}

		req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/topics/%s", adminHostname, topic.Name), nil)
		if err != nil {
			return err
		}
		resp, err := rClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != 204 && resp.StatusCode != 200 {
			return fmt.Errorf("error in delete %s", body)
		}
	}
	return nil
}

func destructureSecret(sec *core.Secret) (string, string, string, string, string, string) {
	username := string(sec.Data["client.id"])
	password := string(sec.Data["client.secret"])
	hostname := string(sec.Data["hostname"])
	adminHostname := string(sec.Data["admin.url"])
	tokenURL := string(sec.Data["token.url"])

	cacert := ""
	if val, ok := sec.Data["cacert"]; ok {
		cacert = string(val)
	}

	return username, password, hostname, adminHostname, tokenURL, cacert
}

func ephemGetTopicName(topic crd.KafkaTopicSpec, env crd.ClowdEnvironment) string {
	return fmt.Sprintf("%s-%s", env.Name, topic.TopicName)
}

func getSecret(p *providers.Provider) (*core.Secret, error) {
	sec := &core.Secret{}
	nn := types.NamespacedName{
		Name:      p.Env.Spec.Providers.Kafka.EphemManagedSecretRef.Name,
		Namespace: p.Env.Spec.Providers.Kafka.EphemManagedSecretRef.Namespace,
	}

	err := p.Client.Get(p.Ctx, nn, sec)

	return sec, err
}

func getTopicList(rClient HTTPClient, adminHostname string, p *providers.Provider) (*TopicsList, error) {
	topicList := &TopicsList{}

	path := url.PathEscape(fmt.Sprintf("size=1000&filter=(env-)?%s.*", p.Env.GetName()))
	url := fmt.Sprintf("%s/api/v1/topics?%s", adminHostname, path)
	resp, err := rClient.Get(url)

	if err != nil {
		return topicList, err
	}

	jsonData, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return topicList, err
	}

	err = json.Unmarshal(jsonData, topicList)
	if err != nil {
		return nil, err
	}

	return topicList, nil
}

func upsertClientCache(username string, password string, tokenURL string, adminHostname string, provider *providers.Provider) HTTPClient {
	httpClient, ok := ClientCache.Get(adminHostname)
	if ok {
		return httpClient
	}
	oauthClientConfig := clientcredentials.Config{
		ClientID:     username,
		ClientSecret: password,
		TokenURL:     tokenURL,
		Scopes:       []string{"openid api.iam.service_accounts"},
	}
	client := ClientCreator(provider, oauthClientConfig)

	ClientCache.Set(adminHostname, client)

	return client
}

func (mep *managedEphemProvider) createConnectSecret() error {
	nn := types.NamespacedName{
		Namespace: getConnectNamespace(mep.Env),
		Name:      fmt.Sprintf("%s-connect", getConnectClusterName(mep.Env)),
	}

	secret := &core.Secret{}
	if err := mep.Cache.Create(EphemKafkaConnectSecret, nn, secret); err != nil {
		return err
	}

	secret.StringData = map[string]string{
		"client.secret": string(mep.secretData["client.secret"]),
	}

	secret.SetOwnerReferences([]metav1.OwnerReference{mep.Env.MakeOwnerReference()})
	secret.SetName(nn.Name)
	secret.SetNamespace(nn.Namespace)
	secret.SetLabels(providers.Labels{"env": mep.Env.Name})

	if err := mep.Cache.Update(EphemKafkaConnectSecret, secret); err != nil {
		return err
	}

	return nil
}

func (mep *managedEphemProvider) configureKafkaConnectCluster() error {

	var err error

	builder := newKafkaConnectBuilder(mep.Provider, mep.secretData)

	err = builder.Create()
	if err != nil {
		return err
	}

	err = builder.VerifyEnvLabel()
	if err != nil {
		return err
	}

	err = builder.BuildSpec()

	if err != nil {
		return fmt.Errorf("could not build spec: %w", err)
	}

	return builder.UpdateCache()
}

func (mep *managedEphemProvider) configureBrokers() error {
	// Look up Kafka cluster's listeners and configuremep.Config.Brokers
	// (we need to know the bootstrap server addresses before provisioning KafkaConnect)

	if err := mep.createConnectSecret(); err != nil {
		return errors.Wrap("failed to create kafka connect cluster secret", err)
	}

	if err := mep.configureKafkaConnectCluster(); err != nil {
		return errors.Wrap("failed to provision kafka connect cluster", err)
	}

	return nil
}

func (mep *managedEphemProvider) configCyndi(app *crd.ClowdApp) error {
	if !app.Spec.Cyndi.Enabled {
		return nil
	}

	if err := createCyndiConfigMap(mep, getConnectNamespace(mep.Env)); err != nil {
		return err
	}

	if err := createCyndiPipeline(mep, app, getConnectNamespace(mep.Env), getConnectClusterName(mep.Env)); err != nil {
		return err
	}

	return nil
}

func (mep *managedEphemProvider) processTopics(app *crd.ClowdApp, httpClient HTTPClient, adminHostname string) error {
	topicConfig := []config.TopicConfig{}

	appList, err := mep.Env.GetAppsInEnv(mep.Ctx, mep.Client)

	if err != nil {
		return errors.Wrap("Topic creation failed: Error listing apps", err)
	}

	for _, topic := range app.Spec.KafkaTopics {
		topicName := ephemGetTopicName(topic, *mep.Env)

		err := mep.ephemProcessTopicValues(mep.Env, app, appList, topic, topicName, httpClient, adminHostname)

		if err != nil {
			return err
		}

		topicConfig = append(
			topicConfig,
			config.TopicConfig{Name: topicName, RequestedName: topic.TopicName},
		)
	}

	mep.Config.Kafka.Topics = topicConfig

	return nil
}

func (mep *managedEphemProvider) getAppTopicTopology(appList *crd.ClowdAppList, topic crd.KafkaTopicSpec) (map[string][]string, []string, []string) {
	keys := map[string][]string{}
	replicaValList := []string{}
	partitionValList := []string{}

	for _, iapp := range appList.Items {
		if iapp.Spec.KafkaTopics != nil {
			for _, itopic := range iapp.Spec.KafkaTopics {
				if itopic.TopicName != topic.TopicName {
					// Only consider a topic that matches the name
					continue
				}
				replicaValList = append(replicaValList, strconv.Itoa(int(itopic.Replicas)))
				partitionValList = append(partitionValList, strconv.Itoa(int(itopic.Partitions)))
				for key := range itopic.Config {
					if _, ok := keys[key]; !ok {
						keys[key] = []string{}
					}
					keys[key] = append(keys[key], itopic.Config[key])
				}
			}
		}
	}

	return keys, replicaValList, partitionValList
}

func (mep *managedEphemProvider) getTopicConfigs(keys map[string][]string) ([]Config, error) {
	topicConfig := []Config{}

	for key, valList := range keys {
		f, ok := conversionMap[key]
		if ok {
			out, _ := f(valList)

			topicConfig = append(topicConfig, Config{
				Key:   key,
				Value: out,
			})
		} else {
			return topicConfig, errors.NewClowderError(fmt.Sprintf("no conversion type for %s", key))
		}
	}

	return topicConfig, nil
}

func (mep *managedEphemProvider) getMaxFromList(list []string, floor int, ceiling int) (int, error) {
	max := 0

	if len(list) > 0 {
		maxValue, err := utils.IntMax(list)
		if err != nil {
			return max, errors.NewClowderError(fmt.Sprintf("could not compute max for %v", list))
		}
		maxValInt, err := strconv.ParseUint(maxValue, 10, 16)
		if err != nil {
			return max, errors.NewClowderError(fmt.Sprintf("could not convert string to int32 for %v", maxValInt))
		}
		max = int(maxValInt)
		if max < 1 {
			max = floor
		} else if ceiling > 0 && ceiling >= floor && max > ceiling {
			max = ceiling
		}
	}
	return max, nil
}

func (mep *managedEphemProvider) getTopicSettings(appList *crd.ClowdAppList, topic crd.KafkaTopicSpec, env *crd.ClowdEnvironment) (Settings, error) {
	settings := Settings{}
	keys, replicaValList, partitionValList := mep.getAppTopicTopology(appList, topic)

	topicConfig, err := mep.getTopicConfigs(keys)
	if err != nil {
		return settings, err
	}

	replicas, err := mep.getMaxFromList(replicaValList, ReplicaNumFloor, ReplicaNumCeiling)
	if err != nil {
		return settings, err
	}

	// Stomp over calculated replica if kafka cluster config is less than 1
	// or the calculated replicas are greater than the kafka cluster config
	if env.Spec.Providers.Kafka.Cluster.Replicas < 1 {
		replicas = 1
	} else if int(env.Spec.Providers.Kafka.Cluster.Replicas) < replicas {
		replicas = int(env.Spec.Providers.Kafka.Cluster.Replicas)
	}

	partitions, err := mep.getMaxFromList(partitionValList, PartitionNumFloor, PartitionNumCeiling)
	if err != nil {
		return settings, err
	}

	settings.NumPartitions = partitions
	settings.NumReplicas = replicas
	settings.Config = topicConfig

	return settings, nil
}

func (mep *managedEphemProvider) getTopicFromKafka(newTopicName string, httpClient HTTPClient, adminHostname string) (*http.Response, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/api/v1/topics/%s", adminHostname, newTopicName))
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return resp, nil
}

func (mep *managedEphemProvider) createTopicOnKafka(newTopicName string, settings Settings, httpClient HTTPClient, adminHostname string) error {
	jsonPayload := JSONPayload{
		Name:     newTopicName,
		Settings: settings,
	}

	buf, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	r := strings.NewReader(string(buf))

	resp, err := httpClient.Post(fmt.Sprintf("%s/api/v1/topics", adminHostname), "application/json", r)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return mep.handleKafkaHTTPError(resp, "bad error status code creating")

}

func (mep *managedEphemProvider) updateTopicOnKafka(newTopicName string, settings Settings, httpClient HTTPClient, adminHostname string) error {
	jsonPayload := settings

	buf, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	r := strings.NewReader(string(buf))

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/api/v1/topics/%s", adminHostname, newTopicName), r)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return mep.handleKafkaHTTPError(resp, "bad error status code updating")
}

func (mep *managedEphemProvider) handleKafkaHTTPError(resp *http.Response, msg string) error {
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyErr, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(fmt.Sprintf("%s %d - %s", msg, resp.StatusCode, bodyErr))
	}
	return nil
}

func (mep *managedEphemProvider) ephemProcessTopicValues(
	env *crd.ClowdEnvironment,
	app *crd.ClowdApp,
	appList *crd.ClowdAppList,
	topic crd.KafkaTopicSpec,
	newTopicName string,
	httpClient HTTPClient,
	adminHostname string,
) error {

	settings, err := mep.getTopicSettings(appList, topic, env)
	if err != nil {
		return err
	}

	resp, err := mep.getTopicFromKafka(newTopicName, httpClient, adminHostname)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		err = mep.createTopicOnKafka(newTopicName, settings, httpClient, adminHostname)
	} else {
		err = mep.updateTopicOnKafka(newTopicName, settings, httpClient, adminHostname)
	}

	return err
}

// Client cache provides a mutex protected cache of http clients
type HTTPClientCache struct {
	cache map[string]HTTPClient
	mutex sync.RWMutex
}

func newHTTPClientCahce() HTTPClientCache {
	return HTTPClientCache{
		cache: map[string]HTTPClient{},
		mutex: sync.RWMutex{},
	}
}

func (cc *HTTPClientCache) Get(hostname string) (HTTPClient, bool) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	val, ok := cc.cache[hostname]
	return val, ok
}

func (cc *HTTPClientCache) Remove(hostname string) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	delete(cc.cache, hostname)
}

func (cc *HTTPClientCache) Set(hostname string, client HTTPClient) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	cc.cache[hostname] = client
}

// ConnectBuilder manages the creation of KafkaConnect resources
type ConnectBuilder struct {
	providers.Provider
	kafkaConnect   *strimzi.KafkaConnect
	namespacedName types.NamespacedName
	secretData     map[string][]byte
}

func newKafkaConnectBuilder(provider providers.Provider, secretData map[string][]byte) ConnectBuilder {
	return ConnectBuilder{
		Provider:   provider,
		secretData: secretData,
	}
}

func (kcb *ConnectBuilder) BuildSpec() error {
	replicas := kcb.getReplicas()
	version := kcb.getVersion()
	image := kcb.getImage()

	config, err := kcb.getSpecConfig()
	if err != nil {
		return fmt.Errorf("could not get config: %w", err)
	}

	requests, err := kcb.getRequests()
	if err != nil {
		return err
	}

	limits, err := kcb.getLimits()
	if err != nil {
		return err
	}

	kcb.kafkaConnect.Spec = &strimzi.KafkaConnectSpec{
		Replicas:         &replicas,
		BootstrapServers: fmt.Sprintf("%s:%s", kcb.getSecret("hostname"), kcb.getSecret("port")),
		Version:          &version,
		Config:           config,
		Image:            &image,
		Resources: &strimzi.KafkaConnectSpecResources{
			Requests: requests,
			Limits:   limits,
		},
		Authentication: &strimzi.KafkaConnectSpecAuthentication{
			ClientId: kcb.getSecretPtr("client.id"),
			ClientSecret: &strimzi.KafkaConnectSpecAuthenticationClientSecret{
				Key:        "client.secret",
				SecretName: fmt.Sprintf("%s-connect", getConnectClusterName(kcb.Env)),
			},
			Type:             "oauth",
			TokenEndpointUri: kcb.getSecretPtr("token.url"),
		},
		Tls: &strimzi.KafkaConnectSpecTls{
			TrustedCertificates: []strimzi.KafkaConnectSpecTlsTrustedCertificatesElem{},
		},
	}
	kcb.setTLSAndAuthentication()
	kcb.setAnnotations()
	return nil
}

func (kcb *ConnectBuilder) Create() error {
	kcb.kafkaConnect = &strimzi.KafkaConnect{}
	err := kcb.Cache.Create(KafkaConnect, kcb.getNamespacedName(), kcb.kafkaConnect)
	return err
}

func (kcb *ConnectBuilder) UpdateCache() error {
	return kcb.Cache.Update(KafkaConnect, kcb.kafkaConnect)
}

// ensure that connect cluster of kcb same name but labelled for different env does not exist
func (kcb *ConnectBuilder) VerifyEnvLabel() error {
	if envLabel, ok := kcb.kafkaConnect.GetLabels()["env"]; ok {
		if envLabel != kcb.Env.Name {
			nn := kcb.getNamespacedName()
			return fmt.Errorf(
				"kafka connect cluster named '%s' found in ns '%s' but tied to env '%s'",
				nn.Name, nn.Namespace, envLabel,
			)
		}
	}
	return nil
}

func (kcb *ConnectBuilder) getSpecConfig() (*apiextensions.JSON, error) {
	var config apiextensions.JSON

	connectClusterConfigs := fmt.Sprintf("%s-connect-cluster-configs", kcb.Env.Name)
	connectClusterOffsets := fmt.Sprintf("%s-connect-cluster-offsets", kcb.Env.Name)
	connectClusterStatus := fmt.Sprintf("%s-connect-cluster-status", kcb.Env.Name)
	connectClusterGroupID := fmt.Sprintf("%s-connect-cluster", kcb.Env.Name)

	err := config.UnmarshalJSON([]byte(fmt.Sprintf(`{
		"config.storage.replication.factor":       "3",
		"config.storage.topic":                    "%s",
		"connector.client.config.override.policy": "All",
		"group.id":                                "%s",
		"offset.storage.replication.factor":       "3",
		"offset.storage.topic":                    "%s",
		"offset.storage.partitions":               "5",
		"status.storage.replication.factor":       "3",
		"status.storage.topic":                    "%s"
	}`, connectClusterConfigs, connectClusterGroupID, connectClusterOffsets, connectClusterStatus)))
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal config: %w", err)
	}

	return &config, nil
}

func (kcb *ConnectBuilder) getLimits() (*apiextensions.JSON, error) {
	return kcb.getResourceSpec(kcb.Env.Spec.Providers.Kafka.Connect.Resources.Limits, `{
        "cpu": "600m",
        "memory": "800Mi"
	}`)
}

func (kcb *ConnectBuilder) getRequests() (*apiextensions.JSON, error) {
	return kcb.getResourceSpec(kcb.Env.Spec.Providers.Kafka.Connect.Resources.Requests, `{
        "cpu": "300m",
        "memory": "500Mi"
	}`)
}

func (kcb *ConnectBuilder) getResourceSpec(field *apiextensions.JSON, defaultJSON string) (*apiextensions.JSON, error) {
	if field != nil {
		return field, nil
	}
	var defaults apiextensions.JSON
	err := defaults.UnmarshalJSON([]byte(defaultJSON))
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal defaults: %w", err)
	}

	return &defaults, nil
}

func (kcb *ConnectBuilder) getNamespacedName() types.NamespacedName {
	if kcb.namespacedName.Name == "" || kcb.kafkaConnect.Namespace == "" {
		kcb.namespacedName = types.NamespacedName{
			Namespace: getConnectNamespace(kcb.Env),
			Name:      getConnectClusterName(kcb.Env),
		}
	}
	return kcb.namespacedName
}

func (kcb *ConnectBuilder) getReplicas() int32 {
	replicas := kcb.Env.Spec.Providers.Kafka.Connect.Replicas
	if replicas < int32(1) {
		replicas = int32(1)
	}
	return replicas
}

func (kcb *ConnectBuilder) getVersion() string {
	version := kcb.Env.Spec.Providers.Kafka.Connect.Version
	if version == "" {
		version = "3.1.0"
	}
	return version
}

func (kcb *ConnectBuilder) getImage() string {
	image := kcb.Env.Spec.Providers.Kafka.Connect.Image
	if image == "" {
		image = DefaultImageKafkaXjoin
	}
	return image
}

func (kcb *ConnectBuilder) getSecret(secret string) string {
	return string(kcb.secretData[secret])
}

func (kcb *ConnectBuilder) getSecretPtr(secret string) *string {
	return utils.StringPtr(kcb.getSecret(secret))
}

func (kcb *ConnectBuilder) setTLSAndAuthentication() {
	if kcb.Env.Spec.Providers.Kafka.EnableLegacyStrimzi {
		return
	}
	username := getConnectClusterUserName(kcb.Env)
	kcb.kafkaConnect.Spec.Tls = &strimzi.KafkaConnectSpecTls{
		TrustedCertificates: []strimzi.KafkaConnectSpecTlsTrustedCertificatesElem{{
			Certificate: "ca.crt",
			SecretName:  fmt.Sprintf("%s-cluster-ca-cert", getKafkaName(kcb.Env)),
		}},
	}
	kcb.kafkaConnect.Spec.Authentication = &strimzi.KafkaConnectSpecAuthentication{
		PasswordSecret: &strimzi.KafkaConnectSpecAuthenticationPasswordSecret{
			Password:   "password",
			SecretName: username,
		},
		Type:     "scram-sha-512",
		Username: &username,
	}
}

func (kcb *ConnectBuilder) setAnnotations() {
	// configures kcb KafkaConnect to use KafkaConnector resources to avoid needing to call the
	// Connect REST API directly
	annotations := kcb.kafkaConnect.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["strimzi.io/use-connector-resources"] = "true"
	kcb.kafkaConnect.SetAnnotations(annotations)
	kcb.kafkaConnect.SetOwnerReferences([]metav1.OwnerReference{kcb.Env.MakeOwnerReference()})
	kcb.kafkaConnect.SetName(getConnectClusterName(kcb.Env))
	kcb.kafkaConnect.SetNamespace(getConnectNamespace(kcb.Env))
	kcb.kafkaConnect.SetLabels(providers.Labels{"env": kcb.Env.Name})
}
