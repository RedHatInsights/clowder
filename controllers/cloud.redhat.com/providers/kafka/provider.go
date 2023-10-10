package kafka

import (
	"fmt"
	"strconv"
	"strings"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	cyndi "github.com/RedHatInsights/cyndi-operator/api/v1alpha1"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	core "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type providerInterface interface {
	providers.RootProvider
	KafkaTopicName(topic crd.KafkaTopicSpec, namespace string) string
	KafkaName() string
	KafkaNamespace() string
	getConnectClusterUserName() string
	getBootstrapServersString() string
	connectConfig(*apiextensions.JSON) error
	getKafkaConnectTrustedCertSecretName() (string, error)
}

type rootKafkaProvider struct {
	providers.Provider
}

var DefaultImageKafkaXjoin = "quay.io/cloudservices/xjoin-kafka-connect-strimzi:latest"

// ProvName is the name/ident of the provider
var ProvName = "kafka"

// CyndiPipeline identifies the main cyndi pipeline object.
var CyndiPipeline = rc.NewSingleResourceIdent(ProvName, "cyndi_pipeline", &cyndi.CyndiPipeline{})

// CyndiAppSecret identifies the cyndi app secret object.
var CyndiAppSecret = rc.NewSingleResourceIdent(ProvName, "cyndi_app_secret", &core.Secret{})

// CyndiHostInventoryAppSecret identifies the cyndi host-inventory app secret object.
var CyndiHostInventoryAppSecret = rc.NewSingleResourceIdent(ProvName, "cyndi_host_inventory_secret", &core.Secret{})

// CyndiConfigMap is the resource ident for a CyndiConfigMap object.
var CyndiConfigMap = rc.NewSingleResourceIdent(ProvName, "cyndi_config_map", &core.ConfigMap{}, rc.ResourceOptions{WriteNow: true})

// KafkaTopic is the resource ident for a KafkaTopic object.
var KafkaTopic = rc.NewSingleResourceIdent(ProvName, "kafka_topic", &strimzi.KafkaTopic{}, rc.ResourceOptions{WriteNow: true})

// KafkaConnect is the resource ident for a KafkaConnect object.
var KafkaConnect = rc.NewSingleResourceIdent(ProvName, "kafka_connect", &strimzi.KafkaConnect{}, rc.ResourceOptions{WriteNow: true})

// GetKafka returns the correct kafka provider based on the environment.
func GetKafka(c *providers.Provider) (providers.ClowderProvider, error) {
	c.Env.ConvertDeprecatedKafkaSpec()
	kafkaMode := c.Env.Spec.Providers.Kafka.Mode
	switch kafkaMode {
	case "operator":
		return NewStrimzi(c)
	case "app-interface":
		return NewAppInterface(c)
	case "managed":
		return NewManagedKafka(c)
	case "ephem-msk":
		return NewMSK(c)
	case "none", "":
		return NewNoneKafka(c)
	default:
		errStr := fmt.Sprintf("No matching kafka mode for %s", kafkaMode)
		return nil, errors.NewClowderError(errStr)
	}
}

func getKafkaUsername(env *crd.ClowdEnvironment, app *crd.ClowdApp) string {
	return fmt.Sprintf("%s-%s", env.Name, app.Name)
}

func getKafkaName(e *crd.ClowdEnvironment) string {
	if e.Spec.Providers.Kafka.Cluster.Name == "" {
		// generate a unique name based on the ClowdEnvironment's UID

		// convert e.UID (which is a apimachinery types.UID) to string
		// types.UID is a string alias so this should not fail...
		uidString := string(e.UID)

		// append the initial portion of the UUID onto the kafka cluster's name
		return fmt.Sprintf("%s-%s", e.Name, strings.Split(uidString, "-")[0])
	}
	return e.Spec.Providers.Kafka.Cluster.Name
}

func getKafkaNamespace(e *crd.ClowdEnvironment) string {
	if e.Spec.Providers.Kafka.Cluster.Namespace == "" {
		return e.Status.TargetNamespace
	}
	return e.Spec.Providers.Kafka.Cluster.Namespace
}

func getConnectNamespace(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Kafka.Connect.Namespace == "" {
		return getKafkaNamespace(env)
	}
	return env.Spec.Providers.Kafka.Connect.Namespace
}

func getConnectClusterName(env *crd.ClowdEnvironment) string {
	if env.Spec.Providers.Kafka.Connect.Name == "" {
		return getKafkaName(env)
	}
	return env.Spec.Providers.Kafka.Connect.Name
}

func init() {
	providers.ProvidersRegistration.Register(GetKafka, 6, ProvName)
}

func processTopics(s providerInterface, app *crd.ClowdApp) error {
	topicConfig := []config.TopicConfig{}

	appList, err := s.GetEnv().GetAppsInEnv(s.GetCtx(), s.GetClient())

	if err != nil {
		return errors.Wrap("Topic creation failed: Error listing apps", err)
	}

	for _, topic := range app.Spec.KafkaTopics {
		k := &strimzi.KafkaTopic{}

		topicName := s.KafkaTopicName(topic, app.Namespace)
		knn := types.NamespacedName{
			Namespace: s.KafkaNamespace(),
			Name:      topicName,
		}

		if err := s.GetCache().Create(KafkaTopic, knn, k); err != nil {
			return err
		}

		labels := providers.Labels{
			"strimzi.io/cluster": s.KafkaName(),
			"env":                app.Spec.EnvName,
			// If we label it with the app name, since app names should be
			// unique? can we use for delete selector?
		}

		k.SetName(topicName)
		k.SetNamespace(s.KafkaNamespace())
		// the ClowdEnvironment is the owner of this topic
		k.SetOwnerReferences([]metav1.OwnerReference{s.GetEnv().MakeOwnerReference()})
		k.SetLabels(labels)

		k.Spec = &strimzi.KafkaTopicSpec{}

		err := processTopicValues(k, s.GetEnv(), appList, topic)

		if err != nil {
			return err
		}

		if err := s.GetCache().Update(KafkaTopic, k); err != nil {
			return err
		}

		topicConfig = append(
			topicConfig,
			config.TopicConfig{Name: topicName, RequestedName: topic.TopicName},
		)
	}

	s.GetConfig().Kafka.Topics = topicConfig

	return nil
}

func processTopicValues(
	k *strimzi.KafkaTopic,
	env *crd.ClowdEnvironment,
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

func configureKafkaConnectCluster(s providerInterface) error {
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
	if s.GetEnv().Spec.Providers.Kafka.Connect.Resources.Requests != nil {
		kcRequests = *s.GetEnv().Spec.Providers.Kafka.Connect.Resources.Requests
	}
	if s.GetEnv().Spec.Providers.Kafka.Connect.Resources.Limits != nil {
		kcLimits = *s.GetEnv().Spec.Providers.Kafka.Connect.Resources.Limits
	}

	clusterNN := types.NamespacedName{
		Namespace: getConnectNamespace(s.GetEnv()),
		Name:      getConnectClusterName(s.GetEnv()),
	}

	k := &strimzi.KafkaConnect{}
	if err := s.GetCache().Create(KafkaConnect, clusterNN, k); err != nil {
		return err
	}

	// ensure that connect cluster of this same name but labelled for different env does not exist
	if envLabel, ok := k.GetLabels()["env"]; ok {
		if envLabel != s.GetEnv().Name {
			return fmt.Errorf(
				"kafka connect cluster named '%s' found in ns '%s' but tied to env '%s'",
				clusterNN.Name, clusterNN.Namespace, envLabel,
			)
		}
	}

	// populate options from the kafka provider's KafkaConnectClusterOptions
	replicas := s.GetEnv().Spec.Providers.Kafka.Connect.Replicas
	if replicas < int32(1) {
		replicas = int32(1)
	}

	version := s.GetEnv().Spec.Providers.Kafka.Connect.Version
	if version == "" {
		version = "3.4.0"
	}

	image := s.GetEnv().Spec.Providers.Kafka.Connect.Image
	if image == "" {
		image = DefaultImageKafkaXjoin
	}

	var config apiextensions.JSON

	err = s.connectConfig(&config)
	if err != nil {
		return fmt.Errorf("could not unmarshal config: %w", err)
	}

	username := s.getConnectClusterUserName()

	k.Spec = &strimzi.KafkaConnectSpec{
		Replicas:         &replicas,
		BootstrapServers: s.getBootstrapServersString(),
		Version:          &version,
		Config:           &config,
		Image:            &image,
		Resources: &strimzi.KafkaConnectSpecResources{
			Requests: &kcRequests,
			Limits:   &kcLimits,
		},
	}

	secName, err := s.getKafkaConnectTrustedCertSecretName()
	if err != nil {
		return err
	}

	if !s.GetEnv().Spec.Providers.Kafka.EnableLegacyStrimzi {
		k.Spec.Tls = &strimzi.KafkaConnectSpecTls{
			TrustedCertificates: []strimzi.KafkaConnectSpecTlsTrustedCertificatesElem{{
				Certificate: "ca.crt",
				SecretName:  secName,
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
	k.SetOwnerReferences([]metav1.OwnerReference{s.GetEnv().MakeOwnerReference()})
	k.SetName(getConnectClusterName(s.GetEnv()))
	k.SetNamespace(getConnectNamespace(s.GetEnv()))
	k.SetLabels(providers.Labels{"env": s.GetEnv().Name})

	return s.GetCache().Update(KafkaConnect, k)
}

func getSecretRef(s providers.RootProvider) (types.NamespacedName, error) {
	secretRef := types.NamespacedName{
		Name:      s.GetEnv().Spec.Providers.Kafka.ManagedSecretRef.Name,
		Namespace: s.GetEnv().Spec.Providers.Kafka.ManagedSecretRef.Namespace,
	}
	nullName := types.NamespacedName{}
	if secretRef == nullName {
		return nullName, errors.NewClowderError("no secret ref defined for managed Kafka")
	}
	return secretRef, nil
}

func getSecret(s providers.RootProvider) (*core.Secret, error) {
	secretRef, err := getSecretRef(s)
	if err != nil {
		return nil, err
	}

	secret := &core.Secret{}

	if err = s.GetClient().Get(s.GetCtx(), secretRef, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func getBrokerConfig(secret *core.Secret) (config.BrokerConfig, error) {
	broker := config.BrokerConfig{}

	port, password, username, hostname, cacert, saslMechanism, err := destructureSecret(secret)
	if err != nil {
		return broker, err
	}

	saslType := config.BrokerConfigAuthtypeSasl

	broker.Hostname = hostname
	broker.Port = &port
	broker.Authtype = &saslType
	if cacert != "" {
		broker.Cacert = &cacert
	}
	broker.Sasl = &config.KafkaSASLConfig{
		Password:         &password,
		Username:         &username,
		SecurityProtocol: utils.StringPtr("SASL_SSL"),
		SaslMechanism:    utils.StringPtr(saslMechanism),
	}
	broker.SecurityProtocol = utils.StringPtr("SASL_SSL")

	return broker, nil
}

func destructureSecret(secret *core.Secret) (int, string, string, string, string, string, error) {
	port, err := strconv.ParseUint(string(secret.Data["port"]), 10, 16)
	if err != nil {
		return 0, "", "", "", "", "", err
	}
	password := string(secret.Data["password"])
	username := string(secret.Data["username"])
	hostname := string(secret.Data["hostname"])
	cacert := ""
	if val, ok := secret.Data["cacert"]; ok {
		cacert = string(val)
	}
	saslMechanism := "PLAIN"
	if val, ok := secret.Data["saslMechanism"]; ok {
		saslMechanism = string(val)
	}
	return int(port), password, username, hostname, cacert, saslMechanism, nil
}
