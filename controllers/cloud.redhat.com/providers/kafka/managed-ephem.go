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
	"github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1/common"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
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
}

type TopicsList struct {
	Items []Topic `json:"items"`
}

type Topic struct {
	Name string `json:"name"`
}

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
			ClientId: common.StringPtr(string(s.secretData["client.id"])),
			ClientSecret: &strimzi.KafkaConnectSpecAuthenticationClientSecret{
				Key:        "client.secret",
				SecretName: fmt.Sprintf("%s-connect", getConnectClusterName(s.Env)),
			},
			Type:             "oauth",
			TokenEndpointUri: common.StringPtr(string(s.secretData["token.url"])),
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
		Config: config.KafkaConfig{
			Brokers: []config.BrokerConfig{{
				Hostname: hostname,
				Port:     utils.IntPtr(443),
				Authtype: &saslType,
				Sasl: &config.KafkaSASLConfig{
					Password:         &password,
					Username:         &username,
					SecurityProtocol: common.StringPtr("SASL_SSL"),
					SaslMechanism:    common.StringPtr("PLAIN"),
				},
			}},
			Topics: []config.TopicConfig{},
		},
		tokenClient:   ReadCache(adminHostname),
		adminHostname: string(sec.Data["admin.url"]),
		secretData:    sec.Data,
	}

	return kafkaProvider, kafkaProvider.configureBrokers()
}

func NewManagedEphemKafkaFinalizer(p *providers.Provider) error {
	if p.Env.Spec.Providers.Kafka.EphemManagedDeletePrefix == "" {
		return nil
	}

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

	if err != nil {
		return err
	}

	jsonData, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	topicList := &TopicsList{}
	json.Unmarshal(jsonData, topicList)

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

func (s *managedEphemProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	if app.Spec.Cyndi.Enabled {
		err := createCyndiPipeline(
			s.Ctx, s.Client, s.Cache, app, s.Env, getConnectNamespace(s.Env), getConnectClusterName(s.Env),
		)
		if err != nil {
			return err
		}
	}

	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	if err := s.processTopics(app); err != nil {
		return err
	}

	// set our provider's config on the AppConfig
	c.Kafka = &s.Config

	return nil
}

func (s *managedEphemProvider) processTopics(app *crd.ClowdApp) error {
	topicConfig := []config.TopicConfig{}

	appList, err := s.Env.GetAppsInEnv(s.Ctx, s.Client)

	if err != nil {
		return errors.Wrap("Topic creation failed: Error listing apps", err)
	}

	for _, topic := range app.Spec.KafkaTopics {
		topicName := ephemGetTopicName(topic, *s.Env)

		err := s.ephemProcessTopicValues(s.Env, app, appList, topic, topicName)

		if err != nil {
			return err
		}

		topicConfig = append(
			topicConfig,
			config.TopicConfig{Name: topicName, RequestedName: topic.TopicName},
		)
	}

	s.Config.Topics = topicConfig

	return nil
}

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
		}
	}

	if env.Spec.Providers.Kafka.Cluster.Replicas < 1 {
		replicas = 1
	} else if int(env.Spec.Providers.Kafka.Cluster.Replicas) < replicas {
		replicas = int(env.Spec.Providers.Kafka.Cluster.Replicas)
	}

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
}
