package kafka

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/pullsecrets"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
)

// KafkaInstance is the resource ident for a Kafka object.
var KafkaInstance = rc.NewSingleResourceIdent(ProvName, "kafka_instance", &strimzi.Kafka{}, rc.ResourceOptions{WriteNow: true})

// KafkaUser is the resource ident for a KafkaUser object.
var KafkaUser = rc.NewSingleResourceIdent(ProvName, "kafka_user", &strimzi.KafkaUser{}, rc.ResourceOptions{WriteNow: true})

// KafkaConnectUser is the resource ident for a KafkaConnectUser object.
var KafkaConnectUser = rc.NewSingleResourceIdent(ProvName, "kafka_connect_user", &strimzi.KafkaUser{}, rc.ResourceOptions{WriteNow: true})

// KafkaMetricsConfigMap is the resource ident for a KafkaMetricsConfigMap object.
var KafkaMetricsConfigMap = rc.NewSingleResourceIdent(ProvName, "kafka_metrics_config_map", &core.ConfigMap{}, rc.ResourceOptions{WriteNow: true})

// KafkaNetworkPolicy is the resource ident for the KafkaNetworkPolicy
var KafkaNetworkPolicy = rc.NewSingleResourceIdent(ProvName, "kafka_network_policy", &networking.NetworkPolicy{}, rc.ResourceOptions{WriteNow: true})

type strimziProvider struct {
	providers.Provider
	rootKafkaProvider
}

// NewStrimzi returns a new strimzi provider object.
func NewStrimzi(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CyndiPipeline,
		CyndiAppSecret,
		CyndiHostInventoryAppSecret,
		CyndiConfigMap,
		KafkaTopic,
		KafkaInstance,
		KafkaConnect,
		KafkaUser,
		KafkaConnectUser,
		KafkaMetricsConfigMap,
		KafkaNetworkPolicy,
	)
	return &strimziProvider{Provider: *p}, nil
}

func (s *strimziProvider) EnvProvide() error {
	if err := createNetworkPolicies(&s.Provider); err != nil {
		return err
	}

	return s.configureBrokers()
}

func (s *strimziProvider) Provide(app *crd.ClowdApp) error {
	clusterNN := types.NamespacedName{
		Namespace: getKafkaNamespace(s.Env),
		Name:      getKafkaName(s.Env),
	}
	kafkaResource := strimzi.Kafka{}
	if err := s.Client.Get(s.Ctx, clusterNN, &kafkaResource); err != nil {
		return err
	}

	kafkaCASecName := types.NamespacedName{
		Name:      fmt.Sprintf("%s-cluster-ca-cert", getKafkaName(s.Env)),
		Namespace: getKafkaNamespace(s.Env),
	}
	kafkaCASecret := core.Secret{}
	if _, err := utils.UpdateOrErr(s.Client.Get(s.Ctx, kafkaCASecName, &kafkaCASecret)); err != nil {
		return err
	}

	_, err := s.HashCache.CreateOrUpdateObject(&kafkaCASecret, true)
	if err != nil {
		return err
	}

	if err = s.HashCache.AddClowdObjectToObject(s.Env, &kafkaCASecret); err != nil {
		return err
	}

	kafkaCACert := string(kafkaCASecret.Data["ca.crt"])

	s.Config.Kafka = &config.KafkaConfig{}
	s.Config.Kafka.Brokers = []config.BrokerConfig{}
	s.Config.Kafka.Topics = []config.TopicConfig{}

	for _, listener := range kafkaResource.Status.Listeners {
		if listener.Name != nil && *listener.Name == "tls" {
			s.Config.Kafka.Brokers = append(s.Config.Kafka.Brokers, buildTLSBrokerConfig(listener, kafkaCACert))
		} else if listener.Name != nil && (*listener.Name == "plain" || *listener.Name == "tcp") {
			s.Config.Kafka.Brokers = append(s.Config.Kafka.Brokers, buildTCPBrokerConfig(listener))
		}
	}

	if app.Spec.Cyndi.Enabled {
		err := createCyndiPipeline(s, app, getConnectNamespace(s.Env), getConnectClusterName(s.Env))
		if err != nil {
			return err
		}
	}

	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	if err := processTopics(s, app); err != nil {
		return err
	}

	if !s.Env.Spec.Providers.Kafka.EnableLegacyStrimzi {
		if err := s.createKafkaUser(app); err != nil {
			return err
		}

		if err := s.setBrokerCredentials(app, s.Config.Kafka); err != nil {
			return err
		}
	}

	return nil
}

var conversionMap = map[string]func([]string) (string, error){
	"retention.ms":          utils.IntMax,
	"retention.bytes":       utils.IntMax,
	"min.compaction.lag.ms": utils.IntMax,
	"cleanup.policy":        utils.ListMerge,
}

func (s *strimziProvider) configureKafkaCluster() error {
	clusterNN := types.NamespacedName{
		Namespace: getKafkaNamespace(s.Env),
		Name:      getKafkaName(s.Env),
	}
	k := &strimzi.Kafka{}
	if err := s.Cache.Create(KafkaInstance, clusterNN, k); err != nil {
		return err
	}

	cmnn, err := s.createKafkaMetricsConfigMap()
	if err != nil {
		return err
	}

	// ensure that kafka cluster of this same name but labelled for different env does not exist
	if envLabel, ok := k.GetLabels()["env"]; ok {
		if envLabel != s.Env.Name {
			return fmt.Errorf(
				"kafka cluster named '%s' found in ns '%s' but tied to env '%s'",
				clusterNN.Name, clusterNN.Namespace, envLabel,
			)
		}
	}

	// populate options from the kafka provider's KafkaClusterOptions
	replicas := s.Env.Spec.Providers.Kafka.Cluster.Replicas
	if replicas < int32(1) {
		replicas = int32(1)
	}

	storageSize := s.Env.Spec.Providers.Kafka.Cluster.StorageSize
	if storageSize == "" {
		storageSize = "1Gi"
	}

	version := s.Env.Spec.Providers.Kafka.Cluster.Version
	if version == "" {
		version = "3.8.0"
	}

	deleteClaim := s.Env.Spec.Providers.Kafka.Cluster.DeleteClaim

	// default values for config/requests/limits in Strimzi resource specs
	var kafConfig, kafRequests, kafLimits, zLimits, zRequests apiextensions.JSON
	var entityUserLimits, entityUserRequests apiextensions.JSON
	var entityTopicLimits, entityTopicRequests apiextensions.JSON
	var entityTLSLimits, entityTLSRequests apiextensions.JSON

	replicasAsStr := strconv.Itoa(int(replicas))
	err = kafConfig.UnmarshalJSON([]byte(fmt.Sprintf(`{
		"offsets.topic.replication.factor": %s,
		"default.replication.factor": %s
	}`, replicasAsStr, replicasAsStr)))
	if err != nil {
		return fmt.Errorf("could not unmarshal kConfig: %w", err)
	}

	err = kafRequests.UnmarshalJSON([]byte(`{
        "cpu": "250m",
        "memory": "600Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal kRequests: %w", err)
	}

	err = kafLimits.UnmarshalJSON([]byte(`{
        "cpu": "500m",
        "memory": "1Gi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal kLimits: %w", err)
	}

	err = zRequests.UnmarshalJSON([]byte(`{
        "cpu": "200m",
        "memory": "400Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal zRequests: %w", err)
	}

	err = zLimits.UnmarshalJSON([]byte(`{
        "cpu": "350m",
        "memory": "800Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal zLimits: %w", err)
	}

	err = entityUserRequests.UnmarshalJSON([]byte(`{
        "cpu": "50m",
        "memory": "250Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal entityUserRequests: %w", err)
	}

	err = entityUserLimits.UnmarshalJSON([]byte(`{
        "cpu": "400m",
        "memory": "500Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal entityUserLimits: %w", err)
	}

	err = entityTopicRequests.UnmarshalJSON([]byte(`{
        "cpu": "50m",
        "memory": "250Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal entityTopicRequests: %w", err)
	}

	err = entityTopicLimits.UnmarshalJSON([]byte(`{
        "cpu": "200m",
        "memory": "500Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal entityTopicLimits: %w", err)
	}

	err = entityTLSRequests.UnmarshalJSON([]byte(`{
        "cpu": "50m",
        "memory": "50Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal entityTlsRequests: %w", err)
	}

	err = entityTLSLimits.UnmarshalJSON([]byte(`{
        "cpu": "100m",
        "memory": "100Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal entityTlsLimits: %w", err)
	}

	// check if defaults have been overridden in ClowdEnvironment
	if s.Env.Spec.Providers.Kafka.Cluster.Resources.Requests != nil {
		kafRequests = *s.Env.Spec.Providers.Kafka.Cluster.Resources.Requests
	}
	if s.Env.Spec.Providers.Kafka.Cluster.Resources.Limits != nil {
		kafLimits = *s.Env.Spec.Providers.Kafka.Cluster.Resources.Limits
	}

	var klabels apiextensions.JSON

	err = klabels.UnmarshalJSON([]byte(`{
        "service" : "strimziKafka"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal klabels: %w", err)
	}

	var useFinalizersEnv []strimzi.KafkaSpecEntityOperatorTemplateTopicOperatorContainerEnvElem

	if clowderconfig.LoadedConfig.Features.DisableStrimziFinalizer {
		useFinalizersEnv = []strimzi.KafkaSpecEntityOperatorTemplateTopicOperatorContainerEnvElem{
			{
				Name:  utils.StringPtr("STRIMZI_USE_FINALIZERS"),
				Value: utils.StringPtr("false"),
			},
		}
	} else {
		useFinalizersEnv = []strimzi.KafkaSpecEntityOperatorTemplateTopicOperatorContainerEnvElem{
			{
				Name:  utils.StringPtr("STRIMZI_USE_FINALIZERS"),
				Value: utils.StringPtr("true"),
			},
		}
	}

	k.Spec = &strimzi.KafkaSpec{
		Kafka: strimzi.KafkaSpecKafka{
			Config:   &kafConfig,
			Version:  &version,
			Replicas: replicas,
			Resources: &strimzi.KafkaSpecKafkaResources{
				Requests: &kafRequests,
				Limits:   &kafLimits,
			},
			Template: &strimzi.KafkaSpecKafkaTemplate{
				PerPodService: &strimzi.KafkaSpecKafkaTemplatePerPodService{
					Metadata: &strimzi.KafkaSpecKafkaTemplatePerPodServiceMetadata{
						Labels: &klabels,
					},
				},
				Pod: &strimzi.KafkaSpecKafkaTemplatePod{
					ImagePullSecrets: []strimzi.KafkaSpecKafkaTemplatePodImagePullSecretsElem{},
					Metadata: &strimzi.KafkaSpecKafkaTemplatePodMetadata{
						Labels: &klabels,
					},
				},
			},
		},
		Zookeeper: strimzi.KafkaSpecZookeeper{
			Replicas: replicas,
			Resources: &strimzi.KafkaSpecZookeeperResources{
				Requests: &zRequests,
				Limits:   &zLimits,
			},
			Template: &strimzi.KafkaSpecZookeeperTemplate{
				NodesService: &strimzi.KafkaSpecZookeeperTemplateNodesService{
					Metadata: &strimzi.KafkaSpecZookeeperTemplateNodesServiceMetadata{
						Labels: &klabels,
					},
				},
				Pod: &strimzi.KafkaSpecZookeeperTemplatePod{
					ImagePullSecrets: []strimzi.KafkaSpecZookeeperTemplatePodImagePullSecretsElem{},
					Metadata: &strimzi.KafkaSpecZookeeperTemplatePodMetadata{
						Labels: &klabels,
					},
				},
			},
		},
		EntityOperator: &strimzi.KafkaSpecEntityOperator{
			Template: &strimzi.KafkaSpecEntityOperatorTemplate{
				Pod: &strimzi.KafkaSpecEntityOperatorTemplatePod{
					Metadata: &strimzi.KafkaSpecEntityOperatorTemplatePodMetadata{
						Labels: &klabels,
					},
					ImagePullSecrets: []strimzi.KafkaSpecEntityOperatorTemplatePodImagePullSecretsElem{},
				},
				TopicOperatorContainer: &strimzi.KafkaSpecEntityOperatorTemplateTopicOperatorContainer{
					Env: useFinalizersEnv,
				},
			},
			TopicOperator: &strimzi.KafkaSpecEntityOperatorTopicOperator{
				Resources: &strimzi.KafkaSpecEntityOperatorTopicOperatorResources{
					Requests: &entityTopicRequests,
					Limits:   &entityTopicLimits,
				},
			},
			UserOperator: &strimzi.KafkaSpecEntityOperatorUserOperator{
				Resources: &strimzi.KafkaSpecEntityOperatorUserOperatorResources{
					Requests: &entityUserRequests,
					Limits:   &entityUserLimits,
				},
			},
		},
	}

	// add pull secrets to the kafka cluster pod template configurations
	secretNames, err := pullsecrets.CopyPullSecrets(&s.Provider, getKafkaNamespace(s.Env), s.Env)

	if err != nil {
		return err
	}

	for _, name := range secretNames {
		k.Spec.Kafka.Template.Pod.ImagePullSecrets = append(k.Spec.Kafka.Template.Pod.ImagePullSecrets, strimzi.KafkaSpecKafkaTemplatePodImagePullSecretsElem{Name: &name})
		k.Spec.Zookeeper.Template.Pod.ImagePullSecrets = append(k.Spec.Zookeeper.Template.Pod.ImagePullSecrets, strimzi.KafkaSpecZookeeperTemplatePodImagePullSecretsElem{Name: &name})
		k.Spec.EntityOperator.Template.Pod.ImagePullSecrets = append(k.Spec.EntityOperator.Template.Pod.ImagePullSecrets, strimzi.KafkaSpecEntityOperatorTemplatePodImagePullSecretsElem{Name: &name})
	}

	if s.Env.Spec.Providers.Kafka.Cluster.Config != nil && len(*s.Env.Spec.Providers.Kafka.Cluster.Config) != 0 {
		jsonData, err := json.Marshal(s.Env.Spec.Providers.Kafka.Cluster.Config)
		if err != nil {
			return err
		}
		err = kafConfig.UnmarshalJSON(jsonData)
		if err != nil {
			return fmt.Errorf("could not unmarshall json data: %w", err)
		}
		k.Spec.Kafka.Config = &kafConfig
	}

	k.Spec.Kafka.JvmOptions = &s.Env.Spec.Providers.Kafka.Cluster.JVMOptions

	metricsConfig := strimzi.KafkaSpecKafkaMetricsConfig{
		Type: "jmxPrometheusExporter",
		ValueFrom: strimzi.KafkaSpecKafkaMetricsConfigValueFrom{
			ConfigMapKeyRef: &strimzi.KafkaSpecKafkaMetricsConfigValueFromConfigMapKeyRef{
				Key:      utils.StringPtr("metrics"),
				Name:     utils.StringPtr(cmnn.Name),
				Optional: utils.FalsePtr(),
			},
		},
	}

	k.Spec.Kafka.MetricsConfig = &metricsConfig

	listener := strimzi.KafkaSpecKafkaListenersElem{
		Type: "internal",
	}

	if s.Env.Spec.Providers.Kafka.EnableLegacyStrimzi {
		listener.Port = 9092
		listener.Tls = false
		listener.Name = "tcp"
	} else {
		listener.Port = 9093
		listener.Tls = true
		listener.Name = "tls"
		listener.Authentication = &strimzi.KafkaSpecKafkaListenersElemAuthentication{
			Type: strimzi.KafkaSpecKafkaListenersElemAuthenticationTypeScramSha512,
		}
		k.Spec.Kafka.Authorization = &strimzi.KafkaSpecKafkaAuthorization{
			Type: strimzi.KafkaSpecKafkaAuthorizationTypeSimple,
		}
	}

	k.Spec.Kafka.Listeners = []strimzi.KafkaSpecKafkaListenersElem{listener}

	if clowderconfig.LoadedConfig.Features.EnableExternalStrimzi {
		externalHost := "localhost"
		externalPort := int32(9094)
		externalListener := strimzi.KafkaSpecKafkaListenersElem{
			Name: "ext",
			Port: 9094,
			Tls:  false,
			Type: "nodeport",
			Configuration: &strimzi.KafkaSpecKafkaListenersElemConfiguration{
				Brokers: []strimzi.KafkaSpecKafkaListenersElemConfigurationBrokersElem{
					{
						AdvertisedHost: &externalHost,
						AdvertisedPort: &externalPort,
						Broker:         0,
					},
				},
			},
		}
		k.Spec.Kafka.Listeners = append(k.Spec.Kafka.Listeners, externalListener)
	}

	if s.Env.Spec.Providers.Kafka.PVC {
		k.Spec.Kafka.Storage = strimzi.KafkaSpecKafkaStorage{
			Type:        strimzi.KafkaSpecKafkaStorageTypePersistentClaim,
			Size:        &storageSize,
			DeleteClaim: &deleteClaim,
		}

		zkStorageSize := "50Gi"

		kafQuantity, err := resource.ParseQuantity(storageSize)

		if err == nil {
			zkQuantity, err := resource.ParseQuantity("50Gi")

			if err == nil && kafQuantity.Cmp(zkQuantity) < 0 {
				// Kafka storage size is less than zkStorageSize
				zkStorageSize = storageSize
			}
		}

		k.Spec.Zookeeper.Storage = strimzi.KafkaSpecZookeeperStorage{
			Type:        strimzi.KafkaSpecZookeeperStorageTypePersistentClaim,
			Size:        &zkStorageSize,
			DeleteClaim: &deleteClaim,
		}
	} else {
		k.Spec.Kafka.Storage = strimzi.KafkaSpecKafkaStorage{
			Type: strimzi.KafkaSpecKafkaStorageTypeEphemeral,
		}
		k.Spec.Zookeeper.Storage = strimzi.KafkaSpecZookeeperStorage{
			Type: strimzi.KafkaSpecZookeeperStorageTypeEphemeral,
		}
	}

	k.SetName(getKafkaName(s.Env))
	k.SetNamespace(getKafkaNamespace(s.Env))
	k.SetLabels(providers.Labels{"env": s.Env.Name})
	k.SetOwnerReferences([]metav1.OwnerReference{s.Env.MakeOwnerReference()})

	return s.Cache.Update(KafkaInstance, k)
}

func (s strimziProvider) connectConfig(config *apiextensions.JSON) error {

	rawConfig := []byte(`{
		"config.storage.replication.factor":       "1",
		"config.storage.topic":                    "connect-cluster-configs",
		"connector.client.config.override.policy": "All",
		"group.id":                                "connect-cluster",
		"offset.storage.replication.factor":       "1",
		"offset.storage.topic":                    "connect-cluster-offsets",
		"status.storage.replication.factor":       "1",
		"status.storage.topic":                    "connect-cluster-status",
		"config.providers":                        "secrets",
		"config.providers.secrets.class":          "io.strimzi.kafka.KubernetesSecretConfigProvider"
	}`)

	return config.UnmarshalJSON(rawConfig)
}

func (s *strimziProvider) getConnectClusterUserName() string {
	return fmt.Sprintf("%s-connect", s.Env.Name)
}

func (s *strimziProvider) createKafkaMetricsConfigMap() (types.NamespacedName, error) {
	cm := &core.ConfigMap{}
	nn := types.NamespacedName{
		Namespace: getKafkaNamespace(s.Env),
		Name:      fmt.Sprintf("%s-metrics", getKafkaName(s.Env)),
	}

	if err := s.Cache.Create(KafkaMetricsConfigMap, nn, cm); err != nil {
		return types.NamespacedName{}, err
	}

	cm.Data = map[string]string{"metrics": string(metricsData)}

	cm.SetName(nn.Name)
	cm.SetNamespace(nn.Namespace)
	cm.SetLabels(providers.Labels{"env": s.Env.Name})
	cm.SetOwnerReferences([]metav1.OwnerReference{s.Env.MakeOwnerReference()})

	if err := s.Cache.Update(KafkaMetricsConfigMap, cm); err != nil {
		return types.NamespacedName{}, err
	}

	return nn, nil
}

func (s *strimziProvider) GetProvider() *providers.Provider {
	return &s.Provider
}

func (s *strimziProvider) getBootstrapServersString() string {
	strArray := []string{}
	for _, bc := range s.Config.Kafka.Brokers {
		if bc.Port != nil {
			strArray = append(strArray, fmt.Sprintf("%s:%d", bc.Hostname, *bc.Port))
		} else {
			strArray = append(strArray, bc.Hostname)
		}
	}
	return strings.Join(strArray, ",")
}

func (s *strimziProvider) getKafkaConnectTrustedCertSecretName() (string, error) {
	return fmt.Sprintf("%s-cluster-ca-cert", getKafkaName(s.GetEnv())), nil
}

func (s *strimziProvider) createKafkaConnectUser() error {

	clusterNN := types.NamespacedName{
		Namespace: getConnectNamespace(s.Env),
		Name:      s.getConnectClusterUserName(),
	}

	ku := &strimzi.KafkaUser{}
	if err := s.Cache.Create(KafkaConnectUser, clusterNN, ku); err != nil {
		return err
	}

	labeler := utils.GetCustomLabeler(
		map[string]string{"strimzi.io/cluster": getKafkaName(s.Env)}, clusterNN, s.Env,
	)

	labeler(ku)

	if s.Env.Spec.Providers.Kafka.EnableLegacyStrimzi {
		ku.Spec = &strimzi.KafkaUserSpec{}
	} else {
		ku.Spec = &strimzi.KafkaUserSpec{
			Authentication: &strimzi.KafkaUserSpecAuthentication{
				Type: strimzi.KafkaUserSpecAuthenticationTypeScramSha512,
			},
			Authorization: &strimzi.KafkaUserSpecAuthorization{
				Acls: []strimzi.KafkaUserSpecAuthorizationAclsElem{},
				Type: strimzi.KafkaUserSpecAuthorizationTypeSimple,
			},
		}

		address := "*"
		topic := "*"
		patternType := strimzi.KafkaUserSpecAuthorizationAclsElemResourcePatternTypeLiteral

		all := strimzi.KafkaUserSpecAuthorizationAclsElemOperationAll

		ku.Spec.Authorization.Acls = append(ku.Spec.Authorization.Acls, strimzi.KafkaUserSpecAuthorizationAclsElem{
			Host:      &address,
			Operation: &all,
			Resource: strimzi.KafkaUserSpecAuthorizationAclsElemResource{
				Name:        &topic,
				PatternType: &patternType,
				Type:        strimzi.KafkaUserSpecAuthorizationAclsElemResourceTypeTopic,
			},
		})

		group := "*"
		ku.Spec.Authorization.Acls = append(ku.Spec.Authorization.Acls, strimzi.KafkaUserSpecAuthorizationAclsElem{
			Host:      &address,
			Operation: &all,
			Resource: strimzi.KafkaUserSpecAuthorizationAclsElemResource{
				Name:        &group,
				PatternType: &patternType,
				Type:        strimzi.KafkaUserSpecAuthorizationAclsElemResourceTypeGroup,
			},
		})
	}

	return s.Cache.Update(KafkaConnectUser, ku)
}

func (s *strimziProvider) configureListeners() error {
	clusterNN := types.NamespacedName{
		Namespace: getKafkaNamespace(s.Env),
		Name:      getKafkaName(s.Env),
	}
	kafkaResource := strimzi.Kafka{}
	if _, err := utils.UpdateOrErr(s.Client.Get(s.Ctx, clusterNN, &kafkaResource)); err != nil {
		return err
	}

	// Return an err if we can't obtain listener status to trigger a requeue in the env controller
	if kafkaResource.Status == nil || kafkaResource.Status.Listeners == nil {
		return fmt.Errorf(
			"kafka cluster '%s' in ns '%s' has no listener status", clusterNN.Name, clusterNN.Namespace,
		)
	}

	// INTERNAL
	kafkaCASecName := types.NamespacedName{
		Name:      fmt.Sprintf("%s-cluster-ca-cert", getKafkaName(s.Env)),
		Namespace: getKafkaNamespace(s.Env),
	}
	kafkaCASecret := core.Secret{}
	if _, err := utils.UpdateOrErr(s.Client.Get(s.Ctx, kafkaCASecName, &kafkaCASecret)); err != nil {
		return err
	}

	kafkaCACert := string(kafkaCASecret.Data["ca.crt"])

	s.Config.Kafka.Brokers = []config.BrokerConfig{}
	for _, listener := range kafkaResource.Status.Listeners {
		if listener.Name != nil && *listener.Name == "tls" {
			s.Config.Kafka.Brokers = append(s.Config.Kafka.Brokers, buildTLSBrokerConfig(listener, kafkaCACert))
		} else if listener.Name != nil && (*listener.Name == "plain" || *listener.Name == "tcp") {
			s.Config.Kafka.Brokers = append(s.Config.Kafka.Brokers, buildTCPBrokerConfig(listener))
		}
	}

	if len(s.Config.Kafka.Brokers) < 1 {
		return fmt.Errorf(
			"kafka cluster '%s' in ns '%s' has no listeners", clusterNN.Name, clusterNN.Namespace,
		)
	}

	return nil
}

func buildTCPBrokerConfig(listener strimzi.KafkaStatusListenersElem) config.BrokerConfig {
	bc := config.BrokerConfig{
		Hostname: *listener.Addresses[0].Host,
	}
	port := listener.Addresses[0].Port
	if port != nil {
		p := int(*port)
		bc.Port = &p
	}
	return bc
}

func buildTLSBrokerConfig(listener strimzi.KafkaStatusListenersElem, caCert string) config.BrokerConfig {
	authType := config.BrokerConfigAuthtypeSasl
	bc := config.BrokerConfig{
		Sasl:     &config.KafkaSASLConfig{},
		Cacert:   &caCert,
		Hostname: *listener.Addresses[0].Host,
		Authtype: &authType,
	}
	port := listener.Addresses[0].Port
	if port != nil {
		p := int(*port)
		bc.Port = &p
	}
	return bc
}

func (s *strimziProvider) configureBrokers() error {
	if err := s.configureKafkaCluster(); err != nil {
		return errors.Wrap("failed to provision kafka cluster", err)
	}

	s.Config = &config.AppConfig{
		Kafka: &config.KafkaConfig{},
	}

	// Look up Kafka cluster's listeners and configure s.Config.Brokers
	// (we need to know the bootstrap server addresses before provisioning KafkaConnect)
	if err := s.configureListeners(); err != nil {
		clowdErr := errors.Wrap("unable to determine kafka broker addresses", err)
		clowdErr.Requeue = true
		return clowdErr
	}

	if err := s.createKafkaConnectUser(); err != nil {
		return err
	}

	if err := configureKafkaConnectCluster(s); err != nil {
		return errors.Wrap("failed to provision kafka connect cluster", err)
	}

	return nil
}

func createNetworkPolicies(p *providers.Provider) error {
	appList, err := p.Env.GetAppsInEnv(p.Ctx, p.Client)
	if err != nil {
		return err
	}

	namespaceSet := map[string]bool{}

	np := &networking.NetworkPolicy{}
	nn := types.NamespacedName{
		Name:      getKafkaName(p.Env),
		Namespace: getKafkaNamespace(p.Env),
	}

	if err := p.Cache.Create(KafkaNetworkPolicy, nn, np); err != nil {
		return err
	}

	npFrom := []networking.NetworkPolicyPeer{}

	for _, app := range appList.Items {
		if _, ok := namespaceSet[app.Namespace]; ok {
			continue
		}

		npFrom = append(npFrom, networking.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": app.Namespace,
				},
			},
		})
		namespaceSet[app.Namespace] = true
	}

	np.Spec.Ingress = []networking.NetworkPolicyIngressRule{{
		From: npFrom,
	}}

	np.Spec.PolicyTypes = []networking.PolicyType{"Ingress"}

	labeler := utils.GetCustomLabeler(nil, nn, p.Env)
	labeler(np)

	return p.Cache.Update(KafkaNetworkPolicy, np)
}

func (s *strimziProvider) setBrokerCredentials(app *crd.ClowdApp, configs *config.KafkaConfig) error {
	for _, broker := range configs.Brokers {
		if broker.Authtype == nil {
			continue
		}
		if *broker.Authtype == config.BrokerConfigAuthtypeSasl {
			ku := &strimzi.KafkaUser{}
			nn := types.NamespacedName{
				Name:      getKafkaUsername(s.Env, app),
				Namespace: getKafkaNamespace(s.Env),
			}

			err := s.Client.Get(s.Ctx, nn, ku)
			if err != nil {
				return err
			}

			if ku.Status == nil || ku.Status.Username == nil {
				return errors.NewClowderError("no username in kafkauser status")
			}
			broker.Sasl.Username = ku.Status.Username

			if ku.Status.Secret == nil {
				return errors.NewClowderError("no secret in kafkauser status")
			}

			secnn := types.NamespacedName{
				Name:      *ku.Status.Secret,
				Namespace: getKafkaNamespace(s.Env),
			}

			kafkaSecret := &core.Secret{}

			err = s.Client.Get(s.Ctx, secnn, kafkaSecret)
			if err != nil {
				return err
			}

			_, err = s.HashCache.CreateOrUpdateObject(kafkaSecret, true)
			if err != nil {
				return err
			}

			if err = s.HashCache.AddClowdObjectToObject(s.Env, kafkaSecret); err != nil {
				return err
			}

			if kafkaSecret.Data["password"] == nil {
				return errors.NewClowderError("no password in kafkauser secret")
			}
			password := string(kafkaSecret.Data["password"])
			broker.Sasl.Password = &password
			broker.Sasl.SecurityProtocol = utils.StringPtr("SASL_SSL")
			broker.Sasl.SaslMechanism = utils.StringPtr("SCRAM-SHA-512")
			broker.SecurityProtocol = utils.StringPtr("SASL_SSL")
		}
	}
	return nil
}

func (s *strimziProvider) createKafkaUser(app *crd.ClowdApp) error {

	ku := &strimzi.KafkaUser{}
	nn := types.NamespacedName{
		Name:      getKafkaUsername(s.Env, app),
		Namespace: getKafkaNamespace(s.Env),
	}

	if err := s.Cache.Create(KafkaUser, nn, ku); err != nil {
		return err
	}
	labeler := utils.GetCustomLabeler(
		map[string]string{"strimzi.io/cluster": getKafkaName(s.Env)}, nn, s.Env,
	)

	labeler(ku)

	ku.Spec = &strimzi.KafkaUserSpec{
		Authentication: &strimzi.KafkaUserSpecAuthentication{
			Type: strimzi.KafkaUserSpecAuthenticationTypeScramSha512,
		},
		Authorization: &strimzi.KafkaUserSpecAuthorization{
			Acls: []strimzi.KafkaUserSpecAuthorizationAclsElem{},
			Type: strimzi.KafkaUserSpecAuthorizationTypeSimple,
		},
	}

	address := "*"
	patternType := strimzi.KafkaUserSpecAuthorizationAclsElemResourcePatternTypeLiteral
	all := strimzi.KafkaUserSpecAuthorizationAclsElemOperationAll

	for _, topic := range app.Spec.KafkaTopics {
		topicName, err := s.KafkaTopicName(topic, app.Namespace)
		if err != nil {
			return err
		}

		ku.Spec.Authorization.Acls = append(ku.Spec.Authorization.Acls, strimzi.KafkaUserSpecAuthorizationAclsElem{
			Host:      &address,
			Operation: &all,
			Resource: strimzi.KafkaUserSpecAuthorizationAclsElemResource{
				Name:        &topicName,
				PatternType: &patternType,
				Type:        strimzi.KafkaUserSpecAuthorizationAclsElemResourceTypeTopic,
			},
		})
	}

	group := "*"
	ku.Spec.Authorization.Acls = append(ku.Spec.Authorization.Acls, strimzi.KafkaUserSpecAuthorizationAclsElem{
		Host:      &address,
		Operation: &all,
		Resource: strimzi.KafkaUserSpecAuthorizationAclsElemResource{
			Name:        &group,
			PatternType: &patternType,
			Type:        strimzi.KafkaUserSpecAuthorizationAclsElemResourceTypeGroup,
		},
	})

	return s.Cache.Update(KafkaUser, ku)
}

func (s *strimziProvider) KafkaTopicName(topic crd.KafkaTopicSpec, namespace ...string) (string, error) {
	if clowderconfig.LoadedConfig.Features.UseComplexStrimziTopicNames {
		if len(namespace) == 0 {
			return "", fmt.Errorf("no namespace passed to topic name call")
		}
		return fmt.Sprintf("%s-%s-%s", topic.TopicName, s.GetEnv().Name, namespace[0]), nil
	}
	return topic.TopicName, nil
}

func (s *strimziProvider) KafkaName() string {
	return getKafkaName(s.Env)
}

func (s *strimziProvider) KafkaNamespace() string {
	return getKafkaNamespace(s.Env)
}
