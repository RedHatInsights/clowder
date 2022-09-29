package kafka

import (
	"context"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	core "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type appInterface struct {
	providers.Provider
}

func (a *appInterface) EnvProvide() error {
	nn := types.NamespacedName{
		Name:      a.Env.Spec.Providers.Kafka.Cluster.Name,
		Namespace: getKafkaNamespace(a.Env),
	}

	err := validateBrokerService(a.Ctx, a.Client, nn)

	if err != nil {
		return err
	}

	return nil
}

func (a *appInterface) Provide(app *crd.ClowdApp) error {
	if app.Spec.Cyndi.Enabled {
		err := validateCyndiPipeline(a.Ctx, a.Client, app, getConnectNamespace(a.Env))
		if err != nil {
			return err
		}
	}

	if len(app.Spec.KafkaTopics) == 0 {
		return nil
	}

	nn := types.NamespacedName{
		Name:      a.Env.Spec.Providers.Kafka.Cluster.Name,
		Namespace: getKafkaNamespace(a.Env),
	}

	a.Config.Kafka = &config.KafkaConfig{
		Topics: []config.TopicConfig{},
		Brokers: []config.BrokerConfig{{
			Hostname: fmt.Sprintf("%v-kafka-bootstrap.%v.svc", nn.Name, nn.Namespace),
			Port:     utils.IntPtr(9092),
		}},
	}

	for _, topic := range app.Spec.KafkaTopics {
		topicName := types.NamespacedName{
			Namespace: getKafkaNamespace(a.Env),
			Name:      topic.TopicName,
		}

		err := validateKafkaTopic(a.Ctx, a.Client, topicName)

		if err != nil {
			return err
		}

		a.Config.Kafka.Topics = append(
			a.Config.Kafka.Topics,
			config.TopicConfig{
				Name:          topic.TopicName,
				RequestedName: topic.TopicName,
			},
		)
	}
	return nil
}

func validateKafkaTopic(ctx context.Context, cl client.Client, nn types.NamespacedName) error {
	if cl == nil {
		// Don't validate topics from within test suite
		return nil
	}

	t := strimzi.KafkaTopic{}
	err := cl.Get(ctx, nn, &t)

	if err != nil {
		missingDeps := errors.MakeMissingDependencies(errors.MissingDependency{
			Source:  "kafka",
			Details: fmt.Sprintf("No KafkaTopic named '%s' found in namespace '%s'", nn.Name, nn.Namespace),
		})
		return &missingDeps
	}

	return nil
}

func validateBrokerService(ctx context.Context, cl client.Client, nn types.NamespacedName) error {
	if cl == nil {
		// Don't validate brokers from within test suite
		return nil
	}

	nn = types.NamespacedName{
		Name:      fmt.Sprintf("%s-kafka-bootstrap", nn.Name),
		Namespace: nn.Namespace,
	}

	svc := core.Service{}
	err := cl.Get(ctx, nn, &svc)

	if err != nil {
		errorText := fmt.Sprintf("Cannot find kafka bootstrap service %s:%s", nn.Namespace, nn.Name)
		newError := errors.New(errorText)
		errors.LogError(ctx, "kafka", newError)
		return newError
	}

	return nil
}

// NewAppInterface returns a new app-interface kafka provider object.
func NewAppInterface(p *providers.Provider) (providers.ClowderProvider, error) {
	return &appInterface{Provider: *p}, nil
}
