package kafka

import (
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

// KafkaProvider is the interface for apps to use to configure kafka topics
type KafkaProvider interface {
	p.Configurable
	CreateTopics(app *crd.ClowdApp) error
}

func GetKafka(c *p.Provider) (KafkaProvider, error) {
	kafkaMode := c.Env.Spec.Providers.Kafka.Mode
	switch kafkaMode {
	case "operator":
		return NewStrimzi(c)
	case "local":
		return NewLocalKafka(c)
	case "app-interface":
		return NewAppInterface(c)
	default:
		errStr := fmt.Sprintf("No matching kafka mode for %s", kafkaMode)
		return nil, errors.New(errStr)
	}
}

func RunAppProvider(provider p.Provider, c *config.AppConfig, app *crd.ClowdApp) error {
	if len(app.Spec.KafkaTopics) != 0 {

		kafkaProvider, err := GetKafka(&provider)

		if err != nil {
			return errors.Wrap("Failed to init kafka provider", err)
		}

		err = kafkaProvider.CreateTopics(app)

		if err != nil {
			return errors.Wrap("Failed to init kafka topic", err)
		}

		kafkaProvider.Configure(c)
	}
	return nil
}

func RunEnvProvider(provider p.Provider) error {
	_, err := GetKafka(&provider)

	if err != nil {
		return err
	}

	return nil
}

func intPtr(i int) *int {
	return &i
}
