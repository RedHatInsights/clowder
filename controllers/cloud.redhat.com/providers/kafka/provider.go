package kafka

import (
	"fmt"

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
	default:
		errStr := fmt.Sprintf("No matching kafka mode for %s", kafkaMode)
		return nil, errors.New(errStr)
	}
}

func intPtr(i int) *int {
	return &i
}

func init() {
	p.ProvidersRegistration.Register(GetKafka, 1, "kafka", true)
}
