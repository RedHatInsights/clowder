package autoscaler

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	keda "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

func makeAutoScalers(deployment *crd.Deployment, app *crd.ClowdApp, c *config.AppConfig, asp *providers.Provider) error {
	s := &keda.ScaledObject{}
	nn := app.GetDeploymentNamespacedName(deployment)
	if err := asp.Cache.Create(CoreAutoScaler, nn, s); err != nil {
		return err
	}

	d := &apps.Deployment{}
	if err := asp.Cache.Get(deployProvider.CoreDeployment, d, nn); err != nil {
		return err
	}

	initAutoScaler(asp.Env, app, d, s, nn, deployment, c)

	return asp.Cache.Update(CoreAutoScaler, s)
}

func ProvideKedaAutoScaler(app *crd.ClowdApp, c *config.AppConfig, asp *providers.Provider, deployment crd.Deployment) error {
	err := makeAutoScalers(&deployment, app, c, asp)
	return err
}

func initAutoScaler(env *crd.ClowdEnvironment, app *crd.ClowdApp, d *apps.Deployment, s *keda.ScaledObject, nn types.NamespacedName, deployment *crd.Deployment, c *config.AppConfig) {
	labels := app.GetLabels()
	labels["pod"] = nn.Name
	app.SetObjectMeta(s, crd.Name(nn.Name), crd.Labels(labels))

	// Set up the watcher to watch the Deployment we created earlier.
	scalerSpec := keda.ScaledObjectSpec{
		ScaleTargetRef:  &keda.ScaleTarget{Name: d.Name, Kind: d.Kind, APIVersion: d.APIVersion},
		PollingInterval: deployment.AutoScaler.PollingInterval,
		CooldownPeriod:  deployment.AutoScaler.CooldownPeriod,
		Advanced:        deployment.AutoScaler.Advanced,
		Fallback:        deployment.AutoScaler.Fallback,
	}

	if deployment.AutoScaler.MinReplicaCount == nil {
		scalerSpec.MinReplicaCount = new(int32)
		*scalerSpec.MinReplicaCount = 1
	} else {
		scalerSpec.MinReplicaCount = deployment.AutoScaler.MinReplicaCount
	}
	if deployment.AutoScaler.MaxReplicaCount == nil {
		scalerSpec.MaxReplicaCount = new(int32)
		*scalerSpec.MaxReplicaCount = 10
	} else {
		scalerSpec.MaxReplicaCount = deployment.AutoScaler.MaxReplicaCount
	}

	triggers := []keda.ScaleTriggers{}
	for _, trigger := range deployment.AutoScaler.Triggers {

		triggerType := getTriggerRoute(trigger.Type, c, env)
		for k, v := range triggerType {
			trigger.Metadata[k] = v
		}
		triggers = append(triggers, trigger)

	}
	scalerSpec.Triggers = triggers

	s.Spec = scalerSpec
}

func getTriggerRoute(triggerType string, c *config.AppConfig, env *crd.ClowdEnvironment) map[string]string {
	result := map[string]string{}
	switch triggerType {
	case "kafka":
		result["bootstrapServers"] = fmt.Sprintf("%s:%d", c.Kafka.Brokers[0].Hostname, *c.Kafka.Brokers[0].Port)
	case "prometheus":
		result["serverAddress"] = env.Status.Prometheus.Hostname

	// The following are the possible triggers for the keda autoscaler.
	// See https://github.com/kedacore/keda/blob/main/pkg/scaling/scale_handler.go#L313.
	// These are here in case we need to pull clowdapp config info into
	// the keda autoscaler.
	case "artemis-queue":
	case "aws-cloudwatch":
	case "aws-kinesis-stream":
	case "aws-sqs-queue":
	case "azure-blob":
	case "azure-eventhub":
	case "azure-log-analytics":
	case "azure-monitor":
	case "azure-pipelines":
	case "azure-queue":
	case "azure-servicebus":
	case "cpu":
	case "cron":
	case "external":
	case "external-push":
	case "gcp-pubsub":
	case "graphite":
	case "huawei-cloudeye":
	case "ibmmq":
	case "influxdb":
	case "kubernetes-workload":
	case "liiklus":
	case "memory":
	case "metrics-api":
	case "mongodb":
	case "mssql":
	case "mysql":
	case "openstack-metric":
	case "openstack-swift":
	case "postgresql":
	case "rabbitmq":
	case "redis":
	case "redis-cluster":
	case "redis-cluster-streams":
	case "redis-streams":
	case "selenium-grid":
	case "solace-event-queue":
	case "stan":
	default:
		return nil
	}
	return result
}
