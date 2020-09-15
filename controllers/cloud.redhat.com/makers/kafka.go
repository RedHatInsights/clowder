package makers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/whippoorwill/v2/apis/kafka.strimzi.io/v1beta1"

	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/utils"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func IntMinMax(listStrInts []string, max bool) (string, error) {
	var listInts []int
	for _, strint := range listStrInts {
		i, err := strconv.Atoi(strint)
		if err != nil {
			return "", err
		}
		listInts = append(listInts, i)
	}
	ol := listInts[0]
	for i, e := range listInts {
		if max {
			if i == 0 || e > ol {
				ol = e
			}
		} else {
			if i == 0 || e < ol {
				ol = e
			}
		}
	}
	return strconv.Itoa(ol), nil
}

func IntMin(listStrInts []string) (string, error) {
	return IntMinMax(listStrInts, false)
}

func IntMax(listStrInts []string) (string, error) {
	return IntMinMax(listStrInts, true)
}

func ListMerge(listStrs []string) (string, error) {
	optionStrings := make(map[string]bool)
	for _, optionsList := range listStrs {
		brokenString := strings.Split(optionsList, ",")
		for _, option := range brokenString {
			optionStrings[strings.TrimSpace(option)] = true
		}
	}
	keys := make([]string, len(optionStrings))

	i := 0
	for key := range optionStrings {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return strings.Join(keys, ","), nil
}

var ConversionMap = map[string]func([]string) (string, error){
	"retention.ms":          IntMax,
	"retention.bytes":       IntMax,
	"min.compaction.lag.ms": IntMax,
	"cleanup.policy":        ListMerge,
}

//KafkaMaker makes the KafkaConfig object
type KafkaMaker struct {
	*Maker
	config config.KafkaConfig
}

//Make function for the KafkaMaker
func (k *KafkaMaker) Make() (ctrl.Result, error) {
	k.config = config.KafkaConfig{}

	var f func() error

	switch k.Base.Spec.Kafka.Provider {
	case "operator":
		f = k.operator
	case "local":
		f = k.local
	}

	if f != nil {
		return ctrl.Result{}, f()
	}

	return ctrl.Result{}, nil
}

//ApplyConfig for the KafkaMaker
func (k *KafkaMaker) ApplyConfig(c *config.AppConfig) {
	c.Kafka = &k.config
}

func (k *KafkaMaker) local() error {
	return nil
}

func (k *KafkaMaker) operator() error {
	if k.App.Spec.KafkaTopics == nil {
		return nil
	}

	k.config.Topics = []config.TopicConfig{}
	k.config.Brokers = []config.BrokerConfig{}

	appList := crd.InsightsAppList{}
	err := k.Client.List(k.Ctx, &appList)

	if err != nil {
		return err
	}

	for _, kafkaTopic := range k.App.Spec.KafkaTopics {
		kRes := strimzi.KafkaTopic{}

		topicName := fmt.Sprintf("%s-%s-%s", kafkaTopic.TopicName, k.Base.Name, k.Request.Namespace)

		kafkaNamespace := types.NamespacedName{
			Namespace: k.Base.Spec.Kafka.Namespace,
			Name:      topicName,
		}

		err := k.Client.Get(k.Ctx, kafkaNamespace, &kRes)
		update, err := utils.UpdateOrErr(err)

		if err != nil {
			return err
		}

		labels := map[string]string{
			"strimzi.io/cluster": k.Base.Spec.Kafka.ClusterName,
			"iapp":               k.App.GetName(),
			// If we label it with the app name, since app names should be
			// unique? can we use for delete selector?
		}

		kRes.SetName(topicName)
		kRes.SetNamespace(k.Base.Spec.Kafka.Namespace)
		kRes.SetLabels(labels)

		kRes.Spec.Replicas = kafkaTopic.Replicas
		kRes.Spec.Partitions = kafkaTopic.Partitions
		kRes.Spec.Config = kafkaTopic.Config

		newConfig := make(map[string]string)

		// This can be improved from an efficiency PoV
		// Loop through all key/value pairs in the config
		for key, value := range kRes.Spec.Config {
			valList := []string{value}
			for _, res := range appList.Items {
				if res.ObjectMeta.Name == k.Request.Name {
					continue
				}
				if res.ObjectMeta.Namespace != k.Request.Namespace {
					continue
				}
				if res.Spec.KafkaTopics != nil {
					for _, topic := range res.Spec.KafkaTopics {
						if topic.Config != nil {
							if val, ok := topic.Config[key]; ok {
								valList = append(valList, val)
							}
						}
					}
				}
			}
			f, ok := ConversionMap[key]
			if ok {
				out, _ := f(valList)
				newConfig[key] = out
			} else {
				err = fmt.Errorf("no conversion type for %s", key)
				return err
			}
		}

		kRes.Spec.Config = newConfig

		err = update.Apply(k.Ctx, k.Client, &kRes)

		if err != nil {
			return err
		}

		k.config.Topics = append(
			k.config.Topics,
			config.TopicConfig{Name: topicName, RequestedName: kafkaTopic.TopicName},
		)
	}

	clusterName := types.NamespacedName{
		Namespace: k.Base.Spec.Kafka.Namespace,
		Name:      k.Base.Spec.Kafka.ClusterName,
	}

	kafkaResource := strimzi.Kafka{}
	err = k.Client.Get(k.Ctx, clusterName, &kafkaResource)

	if err != nil {
		return err
	}

	for _, listener := range kafkaResource.Status.Listeners {
		if listener.Type == "plain" {
			bc := config.BrokerConfig{
				Hostname: listener.Addresses[0].Host,
			}
			port := listener.Addresses[0].Port
			if port != nil {
				p := int(*port)
				bc.Port = &p
			}
			k.config.Brokers = append(k.config.Brokers, bc)
		}
	}

	return nil
}
