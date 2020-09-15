/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	b64 "encoding/base64"

	"go.uber.org/zap"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	crd "cloud.redhat.com/whippoorwill/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/whippoorwill/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/makers"
	// +kubebuilder:scaffold:imports
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var logger *zap.Logger

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	ctrl.SetLogger(ctrlzap.New(ctrlzap.UseDevMode(true)))
	logger, _ = zap.NewProduction()
	defer logger.Sync()
	logger.Info("bootstrapping test environment")

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()

	if err != nil {
		logger.Fatal("Error starting test env", zap.Error(err))
	}

	if cfg == nil {
		logger.Fatal("env config was returned nil")
	}

	err = crd.AddToScheme(clientgoscheme.Scheme)
	err = strimzi.AddToScheme(clientgoscheme.Scheme)

	if err != nil {
		logger.Fatal("Failed to add scheme", zap.Error(err))
	}

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: clientgoscheme.Scheme})

	if err != nil {
		logger.Fatal("Failed to create k8s client", zap.Error(err))
	}

	if k8sClient == nil {
		logger.Fatal("k8sClient was returned nil", zap.Error(err))
	}

	ctx := context.Background()

	nsSpec := &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kafka"}}
	k8sClient.Create(ctx, nsSpec)

	stopManager := make(chan struct{})
	go Run(":8080", false, testEnv.Config, stopManager)
	time.Sleep(5000 * time.Millisecond)
	retCode := m.Run()
	logger.Info("Stopping test env...")
	close(stopManager)
	err = testEnv.Stop()

	if err != nil {
		logger.Fatal("Failed to tear down env", zap.Error(err))
	}
	os.Exit(retCode)
}

func createKafkaCluster() error {
	ctx := context.Background()
	kport := int32(9092)

	cluster := strimzi.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka",
			Namespace: "kafka",
		},
		Status: strimzi.KafkaStatus{
			Listeners: []strimzi.KafkaListener{{
				Type: "plain",
				Addresses: []strimzi.Address{{
					Host: "kafka-boostrap.kafka.svc",
					Port: &kport,
				}},
			}},
		},
	}

	err := k8sClient.Create(ctx, &cluster)

	if err != nil {
		return err
	}

	err = k8sClient.Status().Update(ctx, &cluster)

	if err != nil {
		return err
	}

	return nil
}

func TestCreateInsightsApp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("Creating InsightsApp")

	name := types.NamespacedName{
		Name:      "test",
		Namespace: "default",
	}

	objMeta := metav1.ObjectMeta{
		Name:      name.Name,
		Namespace: name.Namespace,
		Labels: map[string]string{
			"app": "test",
		},
	}

	cwValues := []string{
		"key_id",
		"secret",
		"default",
		"us-east-1",
	}

	cwKeys := []string{
		"aws_access_key_id",
		"aws_secret_access_key",
		"log_group_name",
		"aws_region",
	}

	cwData := map[string][]byte{}

	for i, key := range cwKeys {
		cwData[key] = []byte(b64.StdEncoding.EncodeToString([]byte(cwValues[i])))
	}

	cloudwatch := core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudwatch",
			Namespace: "default",
		},
		Data: cwData,
	}

	err := k8sClient.Create(ctx, &cloudwatch)

	if err != nil {
		t.Error(err)
		return
	}

	err = createKafkaCluster()

	if err != nil {
		t.Error(err)
		return
	}

	ibase := crd.InsightsBase{
		ObjectMeta: objMeta,
		Spec: crd.InsightsBaseSpec{
			Web: crd.WebConfig{
				Port:     int32(8080),
				Provider: "none",
			},
			Metrics: crd.MetricsConfig{
				Port:     int32(9000),
				Path:     "/metrics",
				Provider: "none",
			},
			Kafka: crd.KafkaConfig{
				ClusterName: "kafka",
				Namespace:   "kafka",
				Provider:    "operator",
			},
			Database: crd.DatabaseConfig{
				Image:    "registry.redhat.io/rhel8/postgresql-12:1-36",
				Provider: "local",
			},
			Logging: crd.LoggingConfig{
				Providers: []string{"cloudwatch"},
			},
			ObjectStore: crd.ObjectStoreConfig{
				Provider: "app-interface",
			},
			InMemoryDB: crd.InMemoryDBConfig{
				Provider: "redis",
			},
		},
	}

	replicas := int32(32)
	partitions := int32(5)
	dbVersion := int32(12)

	kafkaTopics := []strimzi.KafkaTopicSpec{
		{
			TopicName:  "inventory",
			Partitions: &partitions,
			Replicas:   &replicas,
		},
	}

	iapp := crd.InsightsApp{
		ObjectMeta: objMeta,
		Spec: crd.InsightsAppSpec{
			Image:       "test:test",
			Base:        ibase.Name,
			KafkaTopics: kafkaTopics,
			Database: crd.InsightsDatabaseSpec{
				Version: &dbVersion,
				Name:    "test",
			},
		},
	}

	err = k8sClient.Create(ctx, &ibase)

	if err != nil {
		t.Error(err)
		return
	}

	// Create InsightsApp
	err = k8sClient.Create(ctx, &iapp)

	if err != nil {
		t.Error(err)
		return
	}

	// See if Deployment is created

	d := apps.Deployment{}

	err = fetchWithDefaults(name, &d)

	if err != nil {
		t.Error(err)
		return
	}

	c := d.Spec.Template.Spec.Containers[0]

	if c.Image != iapp.Spec.Image {
		t.Errorf("Bad image spec %s; expected %s", c.Image, iapp.Spec.Image)
	}

	// See if Secret is mounted

	found := false
	for _, mount := range c.VolumeMounts {
		if mount.Name == "config-secret" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Deployment %s does not have the config volume mounted", d.Name)
		return
	}

	s := core.Service{}

	err = fetchWithDefaults(name, &s)

	if err != nil {
		t.Error(err)
		return
	}

	// Simple test for service right expects there only to be the metrics port
	if len(s.Spec.Ports) > 1 {
		t.Errorf("Bad port count %d; expected 1", len(s.Spec.Ports))
	}

	if s.Spec.Ports[0].Port != ibase.Spec.Metrics.Port {
		t.Errorf("Bad port created %d; expected %d", s.Spec.Ports[0].Port, ibase.Spec.Metrics.Port)
	}

	// Kafka validation

	topic := strimzi.KafkaTopic{}
	topicName := types.NamespacedName{
		Namespace: ibase.Spec.Kafka.Namespace,
		Name:      "inventory",
	}

	err = fetchWithDefaults(topicName, &topic)

	if err != nil {
		t.Error(err)
		return
	}

	if *topic.Spec.Replicas != replicas {
		t.Errorf("Bad topic replica count %d; expected %d", *topic.Spec.Replicas, replicas)
	}
	if *topic.Spec.Partitions != partitions {
		t.Errorf("Bad topic replica count %d; expected %d", *topic.Spec.Partitions, partitions)
	}

	// Secret content validation

	secretConfig := core.Secret{}

	err = k8sClient.Get(ctx, name, &secretConfig)

	if err != nil {
		t.Error(err)
		return
	}

	jsonContent := config.AppConfig{}
	err = json.Unmarshal(secretConfig.Data["cdappconfig.json"], &jsonContent)

	if err != nil {
		t.Error(err)
		return
	}

	cwConfigVals := []string{
		jsonContent.Logging.Cloudwatch.AccessKeyId,
		jsonContent.Logging.Cloudwatch.SecretAccessKey,
		jsonContent.Logging.Cloudwatch.LogGroup,
		jsonContent.Logging.Cloudwatch.Region,
	}

	for i, val := range cwValues {
		if val != cwConfigVals[i] {
			t.Errorf("Wrong cloudwatch config value %s; expected %s", cwConfigVals[i], val)
			return
		}
	}

	if len(jsonContent.Kafka.Brokers[0].Hostname) == 0 {
		t.Error("Kafka broker hostname is not set")
		return
	}

	for i, kafkaTopic := range kafkaTopics {
		actual, expected := jsonContent.Kafka.Topics[i].Name, kafkaTopic.TopicName

		if actual != expected {
			t.Errorf("Wrong topic name %s; expected %s", actual, expected)
		}
	}
}

func TestConverterFuncs(t *testing.T) {
	answer, _ := makers.IntMin([]string{"4", "6", "7"})
	if answer != "4" {
		t.Errorf("Min function should have returned 4, returned %s", answer)
	}

	answer, _ = makers.IntMax([]string{"4", "6", "7"})
	if answer != "7" {
		t.Errorf("Min function should have returned 7, returned %s", answer)
	}

	answer, _ = makers.ListMerge([]string{"4,5,6", "6", "7,2"})
	if answer != "2,4,5,6,7" {
		t.Errorf("Min function should have returned 2,4,5,6,7 returned %s", answer)
	}
}

func fetchWithDefaults(name types.NamespacedName, resource runtime.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return fetch(ctx, name, resource, 20, 100*time.Millisecond)
}

func fetch(ctx context.Context, name types.NamespacedName, resource runtime.Object, retryCount int, sleepTime time.Duration) error {
	var err error

	for i := 1; i <= retryCount; i++ {
		err = k8sClient.Get(ctx, name, resource)

		if err == nil {
			return nil
		} else if !k8serr.IsNotFound(err) {
			return err
		}

		time.Sleep(sleepTime)
	}

	return err
}
