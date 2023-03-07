package kafka

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
)

// KafkaTopic is the resource ident for a KafkaTopic object.
var KafkaTopic = rc.NewSingleResourceIdent(ProvName, "kafka_topic", &strimzi.KafkaTopic{}, rc.ResourceOptions{WriteNow: true})

// KafkaInstance is the resource ident for a Kafka object.
var KafkaInstance = rc.NewSingleResourceIdent(ProvName, "kafka_instance", &strimzi.Kafka{}, rc.ResourceOptions{WriteNow: true})

// KafkaConnect is the resource ident for a KafkaConnect object.
var KafkaConnect = rc.NewSingleResourceIdent(ProvName, "kafka_connect", &strimzi.KafkaConnect{}, rc.ResourceOptions{WriteNow: true})

// KafkaUser is the resource ident for a KafkaUser object.
var KafkaUser = rc.NewSingleResourceIdent(ProvName, "kafka_user", &strimzi.KafkaUser{}, rc.ResourceOptions{WriteNow: true})

// KafkaUser is the resource ident for a KafkaUser object.
var KafkaConnectUser = rc.NewSingleResourceIdent(ProvName, "kafka_connect_user", &strimzi.KafkaUser{}, rc.ResourceOptions{WriteNow: true})

// KafkaMetricsConfigMap is the resource ident for a KafkaMetricsConfigMap object.
var KafkaMetricsConfigMap = rc.NewSingleResourceIdent(ProvName, "kafka_metrics_config_map", &core.ConfigMap{}, rc.ResourceOptions{WriteNow: true})

// KafkaNetworkPolicy is the resource ident for the KafkaNetworkPolicy
var KafkaNetworkPolicy = rc.NewSingleResourceIdent(ProvName, "kafka_network_policy", &networking.NetworkPolicy{}, rc.ResourceOptions{WriteNow: true})

type strimziProvider struct {
	providers.Provider
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

	kafkaCACert := string(kafkaCASecret.Data["ca.crt"])

	s.Config.Kafka = &config.KafkaConfig{}
	s.Config.Kafka.Brokers = []config.BrokerConfig{}
	s.Config.Kafka.Topics = []config.TopicConfig{}

	for _, listener := range kafkaResource.Status.Listeners {
		if listener.Type != nil && *listener.Type == "tls" {
			s.Config.Kafka.Brokers = append(s.Config.Kafka.Brokers, buildTLSBrokerConfig(listener, kafkaCACert))
		} else if listener.Type != nil && (*listener.Type == "plain" || *listener.Type == "tcp") {
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

	if err := s.processTopics(app, s.Config.Kafka); err != nil {
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
		version = "3.1.0"
	}

	deleteClaim := s.Env.Spec.Providers.Kafka.Cluster.DeleteClaim

	// default values for config/requests/limits in Strimzi resource specs
	var kafConfig, kafRequests, kafLimits, zLimits, zRequests apiextensions.JSON
	var entityUserLimits, entityUserRequests apiextensions.JSON
	var entityTopicLimits, entityTopicRequests apiextensions.JSON
	var entityTLSLimits, entityTLSRequests apiextensions.JSON

	err = kafConfig.UnmarshalJSON([]byte(fmt.Sprintf(`{
		"offsets.topic.replication.factor": %s
	}`, strconv.Itoa(int(replicas)))))
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

	k.Spec = &strimzi.KafkaSpec{
		Kafka: strimzi.KafkaSpecKafka{
			Config:   &kafConfig,
			Version:  &version,
			Replicas: replicas,
			Resources: &strimzi.KafkaSpecKafkaResources{
				Requests: &kafRequests,
				Limits:   &kafLimits,
			},
		},
		Zookeeper: strimzi.KafkaSpecZookeeper{
			Replicas: replicas,
			Resources: &strimzi.KafkaSpecZookeeperResources{
				Requests: &zRequests,
				Limits:   &zLimits,
			},
		},
		EntityOperator: &strimzi.KafkaSpecEntityOperator{
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
			TlsSidecar: &strimzi.KafkaSpecEntityOperatorTlsSidecar{
				Resources: &strimzi.KafkaSpecEntityOperatorTlsSidecarResources{
					Requests: &entityTLSRequests,
					Limits:   &entityTLSLimits,
				},
			},
		},
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

	if err := s.Cache.Update(KafkaInstance, k); err != nil {
		return err
	}

	return nil
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

func (s *strimziProvider) getBootstrapServersString(configs *config.KafkaConfig) string {
	strArray := []string{}
	for _, bc := range configs.Brokers {
		if bc.Port != nil {
			strArray = append(strArray, fmt.Sprintf("%s:%d", bc.Hostname, *bc.Port))
		} else {
			strArray = append(strArray, bc.Hostname)
		}
	}
	return strings.Join(strArray, ",")
}

func (s *strimziProvider) createKafkaConnectUser() error {

	clusterNN := types.NamespacedName{
		Namespace: getConnectNamespace(s.Env),
		Name:      getConnectClusterUserName(s.Env),
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

		ku.Spec.Authorization.Acls = append(ku.Spec.Authorization.Acls, strimzi.KafkaUserSpecAuthorizationAclsElem{
			Host:      &address,
			Operation: strimzi.KafkaUserSpecAuthorizationAclsElemOperationAll,
			Resource: strimzi.KafkaUserSpecAuthorizationAclsElemResource{
				Name:        &topic,
				PatternType: &patternType,
				Type:        strimzi.KafkaUserSpecAuthorizationAclsElemResourceTypeTopic,
			},
		})

		group := "*"
		ku.Spec.Authorization.Acls = append(ku.Spec.Authorization.Acls, strimzi.KafkaUserSpecAuthorizationAclsElem{
			Host:      &address,
			Operation: strimzi.KafkaUserSpecAuthorizationAclsElemOperationAll,
			Resource: strimzi.KafkaUserSpecAuthorizationAclsElemResource{
				Name:        &group,
				PatternType: &patternType,
				Type:        strimzi.KafkaUserSpecAuthorizationAclsElemResourceTypeGroup,
			},
		})
	}

	if err := s.Cache.Update(KafkaConnectUser, ku); err != nil {
		return err
	}

	return nil
}

func (s *strimziProvider) configureKafkaConnectCluster(configs *config.KafkaConfig) error {
	var kcRequests, kcLimits apiextensions.JSON

	// default values for config/requests/limits in Strimzi resource specs
	err := kcRequests.UnmarshalJSON([]byte(`{
        "cpu": "300m",
        "memory": "500Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal kcRequests: %w", err)
	}

	err = kcLimits.UnmarshalJSON([]byte(`{
        "cpu": "600m",
        "memory": "800Mi"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal kcLimits: %w", err)
	}

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

	if err := s.createKafkaConnectUser(); err != nil {
		return err
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
		version = "3.1.0"
	}

	image := s.Env.Spec.Providers.Kafka.Connect.Image
	if image == "" {
		image = DefaultImageKafkaXjoin
	}

	username := getConnectClusterUserName(s.Env)

	var config apiextensions.JSON

	err = config.UnmarshalJSON([]byte(`{
		"config.storage.replication.factor":       "1",
		"config.storage.topic":                    "connect-cluster-configs",
		"connector.client.config.override.policy": "All",
		"group.id":                                "connect-cluster",
		"offset.storage.replication.factor":       "1",
		"offset.storage.topic":                    "connect-cluster-offsets",
		"status.storage.replication.factor":       "1",
		"status.storage.topic":                    "connect-cluster-status"
	}`))
	if err != nil {
		return fmt.Errorf("could not unmarshal config: %w", err)
	}

	k.Spec = &strimzi.KafkaConnectSpec{
		Replicas:         &replicas,
		BootstrapServers: s.getBootstrapServersString(configs),
		Version:          &version,
		Config:           &config,
		Image:            &image,
		Resources: &strimzi.KafkaConnectSpecResources{
			Requests: &kcRequests,
			Limits:   &kcLimits,
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

func (s *strimziProvider) configureListeners(configs *config.KafkaConfig) error {
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

	configs.Brokers = []config.BrokerConfig{}
	for _, listener := range kafkaResource.Status.Listeners {
		if listener.Type != nil && *listener.Type == "tls" {
			configs.Brokers = append(configs.Brokers, buildTLSBrokerConfig(listener, kafkaCACert))
		} else if listener.Type != nil && (*listener.Type == "plain" || *listener.Type == "tcp") {
			configs.Brokers = append(configs.Brokers, buildTCPBrokerConfig(listener))
		}
	}

	if len(configs.Brokers) < 1 {
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

	config := &config.KafkaConfig{}

	// Look up Kafka cluster's listeners and configure s.Config.Brokers
	// (we need to know the bootstrap server addresses before provisioning KafkaConnect)
	if err := s.configureListeners(config); err != nil {
		clowdErr := errors.Wrap("unable to determine kafka broker addresses", err)
		clowdErr.Requeue = true
		return clowdErr
	}

	if err := s.configureKafkaConnectCluster(config); err != nil {
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

	if err := p.Cache.Update(KafkaNetworkPolicy, np); err != nil {
		return err
	}

	return nil
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

	for _, topic := range app.Spec.KafkaTopics {
		topicName := getTopicName(topic, *s.Env, app.Namespace)

		ku.Spec.Authorization.Acls = append(ku.Spec.Authorization.Acls, strimzi.KafkaUserSpecAuthorizationAclsElem{
			Host:      &address,
			Operation: strimzi.KafkaUserSpecAuthorizationAclsElemOperationAll,
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
		Operation: strimzi.KafkaUserSpecAuthorizationAclsElemOperationAll,
		Resource: strimzi.KafkaUserSpecAuthorizationAclsElemResource{
			Name:        &group,
			PatternType: &patternType,
			Type:        strimzi.KafkaUserSpecAuthorizationAclsElemResourceTypeGroup,
		},
	})

	if err := s.Cache.Update(KafkaUser, ku); err != nil {
		return err
	}

	return nil
}

func (s *strimziProvider) processTopics(app *crd.ClowdApp, c *config.KafkaConfig) error {
	topicConfig := []config.TopicConfig{}

	appList, err := s.Env.GetAppsInEnv(s.Ctx, s.Client)

	if err != nil {
		return errors.Wrap("Topic creation failed: Error listing apps", err)
	}

	for _, topic := range app.Spec.KafkaTopics {
		k := &strimzi.KafkaTopic{}

		topicName := getTopicName(topic, *s.Env, app.Namespace)
		knn := types.NamespacedName{
			Namespace: getKafkaNamespace(s.Env),
			Name:      topicName,
		}

		if err := s.Cache.Create(KafkaTopic, knn, k); err != nil {
			return err
		}

		labels := providers.Labels{
			"strimzi.io/cluster": getKafkaName(s.Env),
			"env":                app.Spec.EnvName,
			// If we label it with the app name, since app names should be
			// unique? can we use for delete selector?
		}

		k.SetName(topicName)
		k.SetNamespace(getKafkaNamespace(s.Env))
		// the ClowdEnvironment is the owner of this topic
		k.SetOwnerReferences([]metav1.OwnerReference{s.Env.MakeOwnerReference()})
		k.SetLabels(labels)

		k.Spec = &strimzi.KafkaTopicSpec{}

		err := processTopicValues(k, s.Env, app, appList, topic)

		if err != nil {
			return err
		}

		if err := s.Cache.Update(KafkaTopic, k); err != nil {
			return err
		}

		topicConfig = append(
			topicConfig,
			config.TopicConfig{Name: topicName, RequestedName: topic.TopicName},
		)
	}

	c.Topics = topicConfig

	return nil
}

func getTopicName(topic crd.KafkaTopicSpec, env crd.ClowdEnvironment, namespace string) string {
	if clowderconfig.LoadedConfig.Features.UseComplexStrimziTopicNames {
		return fmt.Sprintf("%s-%s-%s", topic.TopicName, env.Name, namespace)
	}
	return topic.TopicName
}

func processTopicValues(
	k *strimzi.KafkaTopic,
	env *crd.ClowdEnvironment,
	app *crd.ClowdApp,
	appList *crd.ClowdAppList,
	topic crd.KafkaTopicSpec,
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

	jsonData := "{"

	for key, valList := range keys {
		f, ok := conversionMap[key]
		if ok {
			out, _ := f(valList)
			jsonData = fmt.Sprintf("%s\"%s\":\"%s\",", jsonData, key, out)
		} else {
			return errors.NewClowderError(fmt.Sprintf("no conversion type for %s", key))
		}
	}

	if len(jsonData) > 1 {
		jsonData = jsonData[0 : len(jsonData)-1]
	}
	jsonData += "}"

	var config apiextensions.JSON

	err := config.UnmarshalJSON([]byte(jsonData))

	if err != nil {
		return err

	}

	k.Spec.Config = &config

	if len(replicaValList) > 0 {
		maxReplicas, err := utils.IntMax(replicaValList)
		if err != nil {
			return errors.NewClowderError(fmt.Sprintf("could not compute max for %v", replicaValList))
		}
		maxReplicasInt, err := strconv.ParseUint(maxReplicas, 10, 16)
		if err != nil {
			return errors.NewClowderError(fmt.Sprintf("could not convert string to int32 for %v", maxReplicas))
		}
		k.Spec.Replicas = utils.Int32Ptr(int(maxReplicasInt))
		if *k.Spec.Replicas < int32(1) {
			// if unset, default to 3
			k.Spec.Replicas = utils.Int32Ptr(3)
		}
	}

	if len(partitionValList) > 0 {
		maxPartitions, err := utils.IntMax(partitionValList)
		if err != nil {
			return errors.NewClowderError(fmt.Sprintf("could not compute max for %v", partitionValList))
		}
		maxPartitionsInt, err := strconv.ParseUint(maxPartitions, 10, 16)
		if err != nil {
			return errors.NewClowderError(fmt.Sprintf("could not convert to string to int32 for %v", maxPartitions))
		}
		k.Spec.Partitions = utils.Int32Ptr(int(maxPartitionsInt))
		if *k.Spec.Partitions < int32(1) {
			// if unset, default to 3
			k.Spec.Partitions = utils.Int32Ptr(3)
		}
	}

	if env.Spec.Providers.Kafka.Cluster.Replicas < int32(1) {
		k.Spec.Replicas = utils.Int32Ptr(1)
	} else if env.Spec.Providers.Kafka.Cluster.Replicas < *k.Spec.Replicas {
		k.Spec.Replicas = &env.Spec.Providers.Kafka.Cluster.Replicas
	}

	return nil
}
