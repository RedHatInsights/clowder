package kafka

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
<<<<<<< HEAD
	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
=======
>>>>>>> @{-1}
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
<<<<<<< HEAD
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
=======

>>>>>>> @{-1}
	core "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
<<<<<<< HEAD
)

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
=======
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// KafkaConnect is the resource ident for a KafkaConnect object.
var EphemKafkaConnect = rc.NewSingleResourceIdent(ProvName, "kafka_connect", &strimzi.KafkaConnect{}, rc.ResourceOptions{WriteNow: true})

// KafkaConnect is the resource ident for a KafkaConnect object.
var EphemKafkaConnectSecret = rc.NewSingleResourceIdent(ProvName, "kafka_connect_secret", &core.Secret{}, rc.ResourceOptions{WriteNow: true})

type managedEphemProvider struct {
	providers.Provider
	Config        config.KafkaConfig
	tokenClient   *http.Client
	adminHostname string
	secretData    map[string][]byte
>>>>>>> @{-1}
}

type TopicsList struct {
	Items []Topic `json:"items"`
}

type Topic struct {
	Name string `json:"name"`
}

<<<<<<< HEAD
// KafkaConnect is the resource ident for a KafkaConnect object.
var EphemKafkaConnect = rc.NewSingleResourceIdent(ProvName, "kafka_connect", &strimzi.KafkaConnect{}, rc.ResourceOptions{WriteNow: true})

var EphemKafkaConnectSecret = rc.NewSingleResourceIdent(ProvName, "kafka_connect_secret", &core.Secret{}, rc.ResourceOptions{WriteNow: true})

//Mutex protected cache of HTTP clients
var ClientCache = newHTTPClientCahce()

const REPLICA_NUM_FLOOR = 3
const REPLICA_NUM_CEILING = 0
const PARTITION_NUM_FLOOR = 3
const PARTITION_NUM_CEILING = 5

func NewManagedEphemKafka(provider *providers.Provider) (providers.ClowderProvider, error) {
	sec, err := getSecret(provider)
	if err != nil {
		return nil, err
	}

	username, password, hostname, adminHostname, tokenURL := destructureSecret(sec)

	httpClient := upsertClientCache(username, password, tokenURL, adminHostname, provider)

	saslType := config.BrokerConfigAuthtypeSasl
	kafkaProvider := &managedEphemProvider{
		Provider: *provider,
=======
func (s *managedEphemProvider) createConnectSecret() error {
	nn := types.NamespacedName{
		Namespace: getConnectNamespace(s.Env),
		Name:      fmt.Sprintf("%s-connect", getConnectClusterName(s.Env)),
	}

	k := &core.Secret{}
	if err := s.Cache.Create(EphemKafkaConnectSecret, nn, k); err != nil {
		return err
	}

	k.StringData = map[string]string{
		"client.secret": string(s.secretData["client.secret"]),
	}

	k.SetOwnerReferences([]metav1.OwnerReference{s.Env.MakeOwnerReference()})
	k.SetName(nn.Name)
	k.SetNamespace(nn.Namespace)
	k.SetLabels(providers.Labels{"env": s.Env.Name})

	if err := s.Cache.Update(EphemKafkaConnectSecret, k); err != nil {
		return err
	}

	return nil
}

func (s *managedEphemProvider) configureKafkaConnectCluster() error {

	//TODO: create secret in namespace as a copy
	var kcRequests, kcLimits apiextensions.JSON

	// default values for config/requests/limits in Strimzi resource specs
	kcRequests.UnmarshalJSON([]byte(`{
        "cpu": "300m",
        "memory": "500Mi"
	}`))

	kcLimits.UnmarshalJSON([]byte(`{
        "cpu": "600m",
        "memory": "800Mi"
	}`))

	// check if defaults have been overridden in ClowdEnvironment
	if s.Env.Spec.Providers.Kafka.Connect.Resources.Requests != nil {
		kcRequests = *s.Env.Spec.Providers.Kafka.Connect.Resources.Requests
	}
	if s.Env.Spec.Providers.Kafka.Connect.Resources.Limits != nil {
		kcLimits = *s.Env.Spec.Providers.Kafka.Connect.Resources.Limits
	}

	clusterNN := types.NamespacedName{
		Namespace: getConnectNamespace(s.Env),
		Name:      getConnectClusterName(s.Env),
	}

	k := &strimzi.KafkaConnect{}
	if err := s.Cache.Create(KafkaConnect, clusterNN, k); err != nil {
		return err
	}

	// ensure that connect cluster of this same name but labelled for different env does not exist
	if envLabel, ok := k.GetLabels()["env"]; ok {
		if envLabel != s.Env.Name {
			return fmt.Errorf(
				"kafka connect cluster named '%s' found in ns '%s' but tied to env '%s'",
				clusterNN.Name, clusterNN.Namespace, envLabel,
			)
		}
	}

	// populate options from the kafka provider's KafkaConnectClusterOptions
	replicas := s.Env.Spec.Providers.Kafka.Connect.Replicas
	if replicas < int32(1) {
		replicas = int32(1)
	}

	version := s.Env.Spec.Providers.Kafka.Connect.Version
	if version == "" {
		version = "3.0.0"
	}

	image := s.Env.Spec.Providers.Kafka.Connect.Image
	if image == "" {
		image = IMAGE_KAFKA_XJOIN
	}

	username := getConnectClusterUserName(s.Env)

	var config apiextensions.JSON

	connectClusterConfigs := fmt.Sprintf("%s-connect-cluster-configs", s.Env.Name)
	connectClusterOffsets := fmt.Sprintf("%s-connect-cluster-offsets", s.Env.Name)
	connectClusterStatus := fmt.Sprintf("%s-connect-cluster-status", s.Env.Name)

	config.UnmarshalJSON([]byte(fmt.Sprintf(`{
		"config.storage.replication.factor":       "1",
		"config.storage.topic":                    "%s",
		"connector.client.config.override.policy": "All",
		"group.id":                                "connect-cluster",
		"offset.storage.replication.factor":       "1",
		"offset.storage.topic":                    "%s",
		"offset.storage.partitions":               "5",
		"status.storage.replication.factor":       "1",
		"status.storage.topic":                    "%s"
	}`, connectClusterConfigs, connectClusterOffsets, connectClusterStatus)))

	k.Spec = &strimzi.KafkaConnectSpec{
		Replicas:         &replicas,
		BootstrapServers: string(s.secretData["hostname"]),
		Version:          &version,
		Config:           &config,
		Image:            &image,
		Resources: &strimzi.KafkaConnectSpecResources{
			Requests: &kcRequests,
			Limits:   &kcLimits,
		},
		Authentication: &strimzi.KafkaConnectSpecAuthentication{
			ClientId: utils.StringPtr(string(s.secretData["client.id"])),
			ClientSecret: &strimzi.KafkaConnectSpecAuthenticationClientSecret{
				Key:        "client.secret",
				SecretName: fmt.Sprintf("%s-connect", getConnectClusterName(s.Env)),
			},
			Type:             "oauth",
			TokenEndpointUri: utils.StringPtr(string(s.secretData["token.url"])),
		},
		Tls: &strimzi.KafkaConnectSpecTls{
			TrustedCertificates: []strimzi.KafkaConnectSpecTlsTrustedCertificatesElem{},
		},
	}

	if !s.Env.Spec.Providers.Kafka.EnableLegacyStrimzi {
		k.Spec.Tls = &strimzi.KafkaConnectSpecTls{
			TrustedCertificates: []strimzi.KafkaConnectSpecTlsTrustedCertificatesElem{{
				Certificate: "ca.crt",
				SecretName:  fmt.Sprintf("%s-cluster-ca-cert", getKafkaName(s.Env)),
			}},
		}
		k.Spec.Authentication = &strimzi.KafkaConnectSpecAuthentication{
			PasswordSecret: &strimzi.KafkaConnectSpecAuthenticationPasswordSecret{
				Password:   "password",
				SecretName: username,
			},
			Type:     "scram-sha-512",
			Username: &username,
		}
	}

	// configures this KafkaConnect to use KafkaConnector resources to avoid needing to call the
	// Connect REST API directly
	annotations := k.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["strimzi.io/use-connector-resources"] = "true"
	k.SetAnnotations(annotations)
	k.SetOwnerReferences([]metav1.OwnerReference{s.Env.MakeOwnerReference()})
	k.SetName(getConnectClusterName(s.Env))
	k.SetNamespace(getConnectNamespace(s.Env))
	k.SetLabels(providers.Labels{"env": s.Env.Name})

	if err := s.Cache.Update(KafkaConnect, k); err != nil {
		return err
	}

	return nil
}

func (s *managedEphemProvider) configureBrokers() error {
	// Look up Kafka cluster's listeners and configure s.Config.Brokers
	// (we need to know the bootstrap server addresses before provisioning KafkaConnect)

	if err := s.createConnectSecret(); err != nil {
		return errors.Wrap("failed to create kafka connect cluster secret", err)
	}

	if err := s.configureKafkaConnectCluster(); err != nil {
		return errors.Wrap("failed to provision kafka connect cluster", err)
	}

	return nil
}

var clientCache = map[string]*http.Client{}

var ccmu sync.RWMutex

func SetCache(hostname string, client *http.Client) {
	ccmu.Lock()
	defer ccmu.Unlock()
	clientCache[hostname] = client
}

func ReleaseCache(hostname string) {
	ccmu.Lock()
	defer ccmu.Unlock()
	delete(clientCache, hostname)
}

func ReadCache(hostname string) *http.Client {
	ccmu.RLock()
	defer ccmu.RUnlock()
	return clientCache[hostname]
}

// NewStrimzi returns a new strimzi provider object.
func NewManagedEphemKafka(p *providers.Provider) (providers.ClowderProvider, error) {
	sec := &core.Secret{}
	nn := types.NamespacedName{
		Name:      p.Env.Spec.Providers.Kafka.EphemManagedSecretRef.Name,
		Namespace: p.Env.Spec.Providers.Kafka.EphemManagedSecretRef.Namespace,
	}

	if err := p.Client.Get(p.Ctx, nn, sec); err != nil {
		return nil, err
	}

	username := string(sec.Data["client.id"])
	password := string(sec.Data["client.secret"])
	hostname := string(sec.Data["hostname"])
	adminHostname := string(sec.Data["admin.url"])

	if _, ok := clientCache[adminHostname]; !ok {
		oauthClientConfig := clientcredentials.Config{
			ClientID:     username,
			ClientSecret: password,
			TokenURL:     string(sec.Data["token.url"]),
			Scopes:       []string{"openid api.iam.service_accounts"},
		}
		client := oauthClientConfig.Client(p.Ctx)

		SetCache(adminHostname, client)
	}

	saslType := config.BrokerConfigAuthtypeSasl
	kafkaProvider := &managedEphemProvider{
		Provider: *p,
>>>>>>> @{-1}
		Config: config.KafkaConfig{
			Brokers: []config.BrokerConfig{{
				Hostname: hostname,
				Port:     utils.IntPtr(443),
				Authtype: &saslType,
				Sasl: &config.KafkaSASLConfig{
					Password:         &password,
					Username:         &username,
<<<<<<< HEAD
					SecurityProtocol: common.StringPtr("SASL_SSL"),
					SaslMechanism:    common.StringPtr("PLAIN"),
=======
					SecurityProtocol: utils.StringPtr("SASL_SSL"),
					SaslMechanism:    utils.StringPtr("PLAIN"),
>>>>>>> @{-1}
				},
			}},
			Topics: []config.TopicConfig{},
		},
<<<<<<< HEAD
		tokenClient:   httpClient,
		adminHostname: adminHostname,
=======
		tokenClient:   ReadCache(adminHostname),
		adminHostname: string(sec.Data["admin.url"]),
>>>>>>> @{-1}
		secretData:    sec.Data,
	}

	return kafkaProvider, kafkaProvider.configureBrokers()
}

func NewManagedEphemKafkaFinalizer(p *providers.Provider) error {
	if p.Env.Spec.Providers.Kafka.EphemManagedDeletePrefix == "" {
		return nil
	}

<<<<<<< HEAD
	sec, err := getSecret(p)
=======
	sec := &core.Secret{}
	nn := types.NamespacedName{
		Name:      p.Env.Spec.Providers.Kafka.EphemManagedSecretRef.Name,
		Namespace: p.Env.Spec.Providers.Kafka.EphemManagedSecretRef.Namespace,
	}

	if err := p.Client.Get(p.Ctx, nn, sec); err != nil {
		return err
	}

	username := string(sec.Data["client.id"])
	password := string(sec.Data["client.secret"])
	adminHostname := string(sec.Data["admin.url"])

	if _, ok := clientCache[adminHostname]; !ok {
		oauthClientConfig := clientcredentials.Config{
			ClientID:     username,
			ClientSecret: password,
			TokenURL:     string(sec.Data["token.url"]),
			Scopes:       []string{"openid api.iam.service_accounts"},
		}
		client := oauthClientConfig.Client(p.Ctx)

		SetCache(adminHostname, client)
	}

	rClient := ReadCache(adminHostname)
	path := url.PathEscape(fmt.Sprintf("size=1000&filter=%s.*", p.Env.GetName()))
	url := fmt.Sprintf("%s/api/v1/topics?%s", adminHostname, path)
	resp, err := rClient.Get(url)

>>>>>>> @{-1}
	if err != nil {
		return err
	}

<<<<<<< HEAD
	username, password, _, adminHostname, tokenURL := destructureSecret(sec)

	rClient := upsertClientCache(username, password, tokenURL, adminHostname, p)

	topicList, err := getTopicList(rClient, adminHostname, p)
=======
	jsonData, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
>>>>>>> @{-1}
	if err != nil {
		return err
	}

<<<<<<< HEAD
	err = deleteTopics(topicList, rClient, adminHostname, p)

	return err
}

func deleteTopics(topicList *TopicsList, rClient *http.Client, adminHostname string, p *providers.Provider) error {
=======
	topicList := &TopicsList{}
	json.Unmarshal(jsonData, topicList)

>>>>>>> @{-1}
	for _, topic := range topicList.Items {
		if strings.HasPrefix(topic.Name, p.Env.Spec.Providers.Kafka.EphemManagedDeletePrefix) {
			req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/topics/%s", adminHostname, topic.Name), nil)
			if err != nil {
				return err
			}
			resp, err := rClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("error in delete %s", body)
			}
		}
	}
	return nil
}

func destructureSecret(sec *core.Secret) (string, string, string, string, string) {
	username := string(sec.Data["client.id"])
	password := string(sec.Data["client.secret"])
	hostname := string(sec.Data["hostname"])
	adminHostname := string(sec.Data["admin.url"])
	tokenURL := string(sec.Data["token.url"])
	return username, password, hostname, adminHostname, tokenURL
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

func getTopicList(rClient *http.Client, adminHostname string, p *providers.Provider) (*TopicsList, error) {
	topicList := &TopicsList{}

	path := url.PathEscape(fmt.Sprintf("size=1000&filter=%s.*", p.Env.GetName()))
	url := fmt.Sprintf("%s/api/v1/topics?%s", adminHostname, path)
	resp, err := rClient.Get(url)

	if err != nil {
		return topicList, err
	}

	jsonData, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return topicList, err
	}

	json.Unmarshal(jsonData, topicList)

	return topicList, nil
}

func upsertClientCache(username string, password string, tokenURL string, adminHostname string, provider *providers.Provider) *http.Client {
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
	client := oauthClientConfig.Client(provider.Ctx)

	ClientCache.Set(adminHostname, client)

	return client
}

type managedEphemProvider struct {
	providers.Provider
	Config        config.KafkaConfig
	tokenClient   *http.Client
	adminHostname string
	secretData    map[string][]byte
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

	builder.BuildSpec()

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

func (mep *managedEphemProvider) createCyndiPipeline(app *crd.ClowdApp) error {
	if !app.Spec.Cyndi.Enabled {
		return nil
	}
	err := createCyndiPipeline(
		mep.Ctx, mep.Client, mep.Cache, app, mep.Env, getConnectNamespace(mep.Env), getConnectClusterName(mep.Env),
	)
	return err
}

func (mep *managedEphemProvider) Provide(app *crd.ClowdApp, appConfig *config.AppConfig) error {

	if err := mep.createCyndiPipeline(app); err != nil {
		return err
	}

	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

<<<<<<< HEAD
	if err := mep.processTopics(app); err != nil {
=======
	if err := s.processTopics(app); err != nil {
>>>>>>> @{-1}
		return err
	}

	// set our provider's config on the AppConfig
<<<<<<< HEAD
	appConfig.Kafka = &mep.Config
=======
	c.Kafka = &s.Config
>>>>>>> @{-1}

	return nil
}

<<<<<<< HEAD
func (mep *managedEphemProvider) processTopics(app *crd.ClowdApp) error {
	topicConfig := []config.TopicConfig{}

	appList, err := mep.Env.GetAppsInEnv(mep.Ctx, mep.Client)
=======
func (s *managedEphemProvider) processTopics(app *crd.ClowdApp) error {
	topicConfig := []config.TopicConfig{}

	appList, err := s.Env.GetAppsInEnv(s.Ctx, s.Client)
>>>>>>> @{-1}

	if err != nil {
		return errors.Wrap("Topic creation failed: Error listing apps", err)
	}

	for _, topic := range app.Spec.KafkaTopics {
<<<<<<< HEAD
		topicName := ephemGetTopicName(topic, *mep.Env)

		err := mep.ephemProcessTopicValues(mep.Env, app, appList, topic, topicName)
=======
		topicName := ephemGetTopicName(topic, *s.Env)

		err := s.ephemProcessTopicValues(s.Env, app, appList, topic, topicName)
>>>>>>> @{-1}

		if err != nil {
			return err
		}

		topicConfig = append(
			topicConfig,
			config.TopicConfig{Name: topicName, RequestedName: topic.TopicName},
		)
	}

<<<<<<< HEAD
	mep.Config.Topics = topicConfig
=======
	s.Config.Topics = topicConfig
>>>>>>> @{-1}

	return nil
}

<<<<<<< HEAD
func (mep *managedEphemProvider) getAppTopicTopology(appList *crd.ClowdAppList, topic crd.KafkaTopicSpec) (map[string][]string, []string, []string) {
=======
func ephemGetTopicName(topic crd.KafkaTopicSpec, env crd.ClowdEnvironment) string {
	return fmt.Sprintf("%s-%s", env.Name, topic.TopicName)
}

func (s *managedEphemProvider) ephemProcessTopicValues(
	env *crd.ClowdEnvironment,
	app *crd.ClowdApp,
	appList *crd.ClowdAppList,
	topic crd.KafkaTopicSpec,
	newTopicName string,
) error {

>>>>>>> @{-1}
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

<<<<<<< HEAD
	return keys, replicaValList, partitionValList
}

func (mep *managedEphemProvider) getTopicConfigs(keys map[string][]string) ([]Config, error) {
=======
>>>>>>> @{-1}
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
<<<<<<< HEAD
			return topicConfig, errors.New(fmt.Sprintf("no conversion type for %s", key))
		}
	}

	return topicConfig, nil
}

func (mep *managedEphemProvider) getMaxFromList(list []string, floor int, ceiling int) (int, error) {
	max := 0

	if len(list) > 0 {
		maxValue, err := utils.IntMax(list)
		if err != nil {
			return max, errors.New(fmt.Sprintf("could not compute max for %v", list))
		}
		maxValInt, err := strconv.Atoi(maxValue)
		if err != nil {
			return max, errors.New(fmt.Sprintf("could not convert string to int32 for %v", maxValInt))
		}
		max = maxValInt
		if max < 1 {
			max = floor
		} else if ceiling > 0 && ceiling > floor && max > ceiling {
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

	replicas, err := mep.getMaxFromList(replicaValList, REPLICA_NUM_FLOOR, REPLICA_NUM_CEILING)
	if err != nil {
		return settings, err
	}

	//Stomp over calculated replica if kafka cluster config is less than 1
	//or the calculated replicas are greater than the kafka cluster config
=======
			return errors.New(fmt.Sprintf("no conversion type for %s", key))
		}
	}

	var replicas int
	var partitions int

	if len(replicaValList) > 0 {
		maxReplicas, err := utils.IntMax(replicaValList)
		if err != nil {
			return errors.New(fmt.Sprintf("could not compute max for %v", replicaValList))
		}
		maxReplicasInt, err := strconv.Atoi(maxReplicas)
		if err != nil {
			return errors.New(fmt.Sprintf("could not convert string to int32 for %v", maxReplicas))
		}
		replicas = maxReplicasInt
		if replicas < 1 {
			// if unset, default to 3
			replicas = 3
		}
	}

	if len(partitionValList) > 0 {
		maxPartitions, err := utils.IntMax(partitionValList)
		if err != nil {
			return errors.New(fmt.Sprintf("could not compute max for %v", partitionValList))
		}
		maxPartitionsInt, err := strconv.Atoi(maxPartitions)
		if err != nil {
			return errors.New(fmt.Sprintf("could not convert to string to int32 for %v", maxPartitions))
		}
		partitions = maxPartitionsInt
		if partitions < 1 {
			// if unset, default to 3
			partitions = 3
		} else if partitions > 5 {
			partitions = 5
		}
	}

>>>>>>> @{-1}
	if env.Spec.Providers.Kafka.Cluster.Replicas < 1 {
		replicas = 1
	} else if int(env.Spec.Providers.Kafka.Cluster.Replicas) < replicas {
		replicas = int(env.Spec.Providers.Kafka.Cluster.Replicas)
	}

<<<<<<< HEAD
	partitions, err := mep.getMaxFromList(partitionValList, PARTITION_NUM_FLOOR, PARTITION_NUM_CEILING)
	if err != nil {
		return settings, err
	}

	settings.NumPartitions = partitions
	settings.NumReplicas = replicas
	settings.Config = topicConfig

	return settings, nil
}

func (mep *managedEphemProvider) getTopicFromKafka(newTopicName string) (*http.Response, error) {
	resp, err := mep.tokenClient.Get(fmt.Sprintf("%s/api/v1/topics/%s", mep.adminHostname, newTopicName))
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return resp, nil
}

func (mep *managedEphemProvider) createTopicOnKafka(newTopicName string, settings Settings) error {
	jsonPayload := JSONPayload{
		Name:     newTopicName,
		Settings: settings,
	}

	buf, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	r := strings.NewReader(string(buf))

	resp, err := mep.tokenClient.Post(fmt.Sprintf("%s/api/v1/topics", mep.adminHostname), "application/json", r)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return mep.handleKafkaHTTPError(resp, "bad error status code creating")

}

func (mep *managedEphemProvider) updateTopicOnKafka(newTopicName string, settings Settings) error {
	jsonPayload := settings

	buf, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	r := strings.NewReader(string(buf))

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/api/v1/topics/%s", mep.adminHostname, newTopicName), r)
	if err != nil {
		return err
	}

	resp, err := mep.tokenClient.Do(req)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return mep.handleKafkaHTTPError(resp, "bad error status code updating")
}

func (mep *managedEphemProvider) handleKafkaHTTPError(resp *http.Response, msg string) error {
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyErr, _ := ioutil.ReadAll(resp.Body)
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
) error {

	settings, err := mep.getTopicSettings(appList, topic, env)
	if err != nil {
		return err
	}

	resp, err := mep.getTopicFromKafka(newTopicName)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		err = mep.createTopicOnKafka(newTopicName, settings)
	} else {
		err = mep.updateTopicOnKafka(newTopicName, settings)
	}

	return err
}

//Client cache provides a mutex protected cache of http clients
type HTTPClientCache struct {
	cache map[string]*http.Client
	mutex sync.RWMutex
}

func newHTTPClientCahce() HTTPClientCache {
	return HTTPClientCache{
		cache: map[string]*http.Client{},
		mutex: sync.RWMutex{},
	}
}

func (cc *HTTPClientCache) Get(hostname string) (*http.Client, bool) {
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

func (cc *HTTPClientCache) Set(hostname string, client *http.Client) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	cc.cache[hostname] = client
}

//KafkaConnectBuilder manages the creation of KafkaConnect resources
type KafkaConnectBuilder struct {
	providers.Provider
	kafkaConnect   *strimzi.KafkaConnect
	namespacedName types.NamespacedName
	secretData     map[string][]byte
}

func newKafkaConnectBuilder(provider providers.Provider, secretData map[string][]byte) KafkaConnectBuilder {
	return KafkaConnectBuilder{
		Provider:   provider,
		secretData: secretData,
	}
}

func (kcb *KafkaConnectBuilder) BuildSpec() {
	replicas := kcb.getReplicas()
	version := kcb.getVersion()
	image := kcb.getImage()
	kcb.kafkaConnect.Spec = &strimzi.KafkaConnectSpec{
		Replicas:         &replicas,
		BootstrapServers: kcb.getSecret("hostname"),
		Version:          &version,
		Config:           kcb.getSpecConfig(),
		Image:            &image,
		Resources: &strimzi.KafkaConnectSpecResources{
			Requests: kcb.getRequests(),
			Limits:   kcb.getLimits(),
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
}

func (kcb *KafkaConnectBuilder) Create() error {
	kcb.kafkaConnect = &strimzi.KafkaConnect{}
	err := kcb.Cache.Create(KafkaConnect, kcb.getNamespacedName(), kcb.kafkaConnect)
	return err
}

func (kcb *KafkaConnectBuilder) UpdateCache() error {
	return kcb.Cache.Update(KafkaConnect, kcb.kafkaConnect)
}

// ensure that connect cluster of kcb same name but labelled for different env does not exist
func (kcb *KafkaConnectBuilder) VerifyEnvLabel() error {
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

func (kcb *KafkaConnectBuilder) getSpecConfig() *apiextensions.JSON {
	var config apiextensions.JSON

	connectClusterConfigs := fmt.Sprintf("%s-connect-cluster-configs", kcb.Env.Name)
	connectClusterOffsets := fmt.Sprintf("%s-connect-cluster-offsets", kcb.Env.Name)
	connectClusterStatus := fmt.Sprintf("%s-connect-cluster-status", kcb.Env.Name)

	config.UnmarshalJSON([]byte(fmt.Sprintf(`{
		"config.storage.replication.factor":       "1",
		"config.storage.topic":                    "%s",
		"connector.client.config.override.policy": "All",
		"group.id":                                "connect-cluster",
		"offset.storage.replication.factor":       "1",
		"offset.storage.topic":                    "%s",
		"status.storage.replication.factor":       "1",
		"status.storage.topic":                    "%s"
	}`, connectClusterConfigs, connectClusterOffsets, connectClusterStatus)))

	return &config
}

func (kcb *KafkaConnectBuilder) getLimits() *apiextensions.JSON {
	return kcb.getResourceSpec(kcb.Env.Spec.Providers.Kafka.Connect.Resources.Limits, `{
        "cpu": "600m",
        "memory": "800Mi"
	}`)
}

func (kcb *KafkaConnectBuilder) getRequests() *apiextensions.JSON {
	return kcb.getResourceSpec(kcb.Env.Spec.Providers.Kafka.Connect.Resources.Requests, `{
        "cpu": "300m",
        "memory": "500Mi"
	}`)
}

func (kcb *KafkaConnectBuilder) getResourceSpec(field *apiextensions.JSON, defaultJSON string) *apiextensions.JSON {
	if field != nil {
		return field
	}
	var defaults apiextensions.JSON
	defaults.UnmarshalJSON([]byte(defaultJSON))

	return &defaults
}

func (kcb *KafkaConnectBuilder) getNamespacedName() types.NamespacedName {
	if kcb.namespacedName.Name == "" || kcb.kafkaConnect.Namespace == "" {
		kcb.namespacedName = types.NamespacedName{
			Namespace: getConnectNamespace(kcb.Env),
			Name:      getConnectClusterName(kcb.Env),
		}
	}
	return kcb.namespacedName
}

func (kcb *KafkaConnectBuilder) getReplicas() int32 {
	replicas := kcb.Env.Spec.Providers.Kafka.Connect.Replicas
	if replicas < int32(1) {
		replicas = int32(1)
	}
	return replicas
}

func (kcb *KafkaConnectBuilder) getVersion() string {
	version := kcb.Env.Spec.Providers.Kafka.Connect.Version
	if version == "" {
		version = "3.0.0"
	}
	return version
}

func (kcb *KafkaConnectBuilder) getImage() string {
	image := kcb.Env.Spec.Providers.Kafka.Connect.Image
	if image == "" {
		image = IMAGE_KAFKA_XJOIN
	}
	return image
}

func (kcb *KafkaConnectBuilder) getSecret(secret string) string {
	return string(kcb.secretData[secret])
}

func (kcb *KafkaConnectBuilder) getSecretPtr(secret string) *string {
	return common.StringPtr(kcb.getSecret(secret))
}

func (kcb *KafkaConnectBuilder) setTLSAndAuthentication() {
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

func (kcb *KafkaConnectBuilder) setAnnotations() {
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
=======
	settings := Settings{
		NumPartitions: int(partitions),
		NumReplicas:   int(replicas),
		Config:        topicConfig,
	}

	resp, err := s.tokenClient.Get(fmt.Sprintf("%s/api/v1/topics/%s", s.adminHostname, newTopicName))

	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == 404 {
		jsonPayload := JSONPayload{
			Name:     newTopicName,
			Settings: settings,
		}

		buf, err := json.Marshal(jsonPayload)
		if err != nil {
			return err
		}

		r := strings.NewReader(string(buf))

		resp, err := s.tokenClient.Post(fmt.Sprintf("%s/api/v1/topics", s.adminHostname), "application/json", r)

		if err != nil {
			return err
		}

		resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			bodyErr, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf(fmt.Sprintf("bad error status code creating %d - %s", resp.StatusCode, bodyErr))
		}
	} else {
		jsonPayload := settings

		buf, err := json.Marshal(jsonPayload)
		if err != nil {
			return err
		}

		r := strings.NewReader(string(buf))

		req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/api/v1/topics/%s", s.adminHostname, newTopicName), r)
		if err != nil {
			return err
		}

		resp, err := s.tokenClient.Do(req)

		if err != nil {
			return err
		}

		resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			bodyErr, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf(fmt.Sprintf("bad error status code updating %d - %s", resp.StatusCode, bodyErr))
		}
	}

	return nil
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
>>>>>>> @{-1}
}
