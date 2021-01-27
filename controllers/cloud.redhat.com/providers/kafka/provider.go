package kafka

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

func GetKafka(c *p.Provider) (p.ClowderProvider, error) {
	kafkaMode := c.Env.Spec.Providers.Kafka.Mode
	switch kafkaMode {
	case "operator":
		return NewStrimzi(c)
	case "local":
		return NewLocalKafka(c)
	case "app-interface":
		return NewAppInterface(c)
	case "none":
		return NewNoneKafka(c)
	default:
		errStr := fmt.Sprintf("No matching kafka mode for %s", kafkaMode)
		return nil, errors.New(errStr)
	}
}

func getKafkaNamespace(e *crd.ClowdEnvironment) string {
	return e.Spec.Providers.Kafka.Namespace
}

func getConnectNamespace(env *crd.ClowdEnvironment, defaultValue string) string {
	if env.Spec.Providers.Kafka.ConnectNamespace == "" {
		return defaultValue
	}
	return env.Spec.Providers.Kafka.ConnectNamespace
}

func getConnectClusterName(env *crd.ClowdEnvironment, defaultValue string) string {
	if env.Spec.Providers.Kafka.ConnectClusterName == "" {
		return defaultValue
	}
	return env.Spec.Providers.Kafka.ConnectClusterName
}

func init() {
	p.ProvidersRegistration.Register(GetKafka, 1, "kafka")
}
