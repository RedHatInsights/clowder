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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
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

	for i := 1; i <= 50; i++ {
		resp, err := http.Get("http://localhost:8080/metrics")

		if err == nil && resp.StatusCode == 200 {
			logger.Info("Manager ready", zap.Int("duration", 100*i))
			break
		}

		if i == 50 {
			if err != nil {
				logger.Fatal("Failed to fetch to metrics for manager after 5s", zap.Error(err))
			}

			logger.Fatal("Failed to get 200 result for metrics", zap.Int("status", resp.StatusCode))
		}

		time.Sleep(100 * time.Millisecond)
	}

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

func createCloudwatchSecret(cwData *map[string]string) error {
	cloudwatch := core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudwatch",
			Namespace: "default",
		},
		StringData: *cwData,
	}

	return k8sClient.Create(context.Background(), &cloudwatch)
}

func createCRs(name types.NamespacedName) (*crd.ClowdEnvironment, *crd.ClowdApp, error) {
	ctx := context.Background()

	objMeta := metav1.ObjectMeta{
		Name:      name.Name,
		Namespace: name.Namespace,
		Labels: map[string]string{
			"app": "test",
		},
	}

	env := crd.ClowdEnvironment{
		ObjectMeta: objMeta,
		Spec: crd.ClowdEnvironmentSpec{
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
				Provider: "app-interface",
			},
			ObjectStore: crd.ObjectStoreConfig{
				Provider: "app-interface",
			},
			InMemoryDB: crd.InMemoryDBConfig{
				Provider: "redis",
			},
			Namespace: objMeta.Namespace,
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

	app := crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Pods: []crd.PodSpec{{
				Image: "test:test",
				Name:  "testpod",
			}},
			EnvName:     env.Name,
			KafkaTopics: kafkaTopics,
			Database: crd.DatabaseSpec{
				Version: &dbVersion,
				Name:    "test",
			},
		},
	}

	err := k8sClient.Create(ctx, &env)

	if err != nil {
		return &env, &app, err
	}

	err = k8sClient.Create(ctx, &app)

	return &env, &app, err
}

func fetchConfig(name types.NamespacedName) (*config.AppConfig, error) {

	secretConfig := core.Secret{}
	jsonContent := config.AppConfig{}

	err := k8sClient.Get(context.Background(), name, &secretConfig)

	if err != nil {
		return &jsonContent, err
	}

	err = json.Unmarshal(secretConfig.Data["cdappconfig.json"], &jsonContent)

	return &jsonContent, err
}

func TestCreateClowdApp(t *testing.T) {
	logger.Info("Creating ClowdApp")

	name := types.NamespacedName{
		Name:      "test",
		Namespace: "default",
	}

	cwData := map[string]string{
		"aws_access_key_id":     "key_id",
		"aws_secret_access_key": "secret",
		"log_group_name":        "default",
		"aws_region":            "us-east-1",
	}

	err := createCloudwatchSecret(&cwData)

	if err != nil {
		t.Error(err)
		return
	}

	err = createKafkaCluster()

	if err != nil {
		t.Error(err)
		return
	}

	env, app, err := createCRs(name)

	if err != nil {
		t.Error(err)
		return
	}

	// See if Deployment is created

	d := apps.Deployment{}

	appnn := types.NamespacedName{
		Name:      app.Spec.Pods[0].Name,
		Namespace: name.Namespace,
	}
	err = fetchWithDefaults(appnn, &d)

	if err != nil {
		t.Error(err)
		return
	}

	antiAffinity := d.Spec.Template.Spec.Affinity.PodAntiAffinity
	terms := antiAffinity.PreferredDuringSchedulingIgnoredDuringExecution

	if len(terms) != 2 {
		t.Errorf("Incorrect number of anti-affinity terms: %d; expected 2", len(terms))
	}

	c := d.Spec.Template.Spec.Containers[0]

	if c.Image != app.Spec.Pods[0].Image {
		t.Errorf("Bad image spec %s; expected %s", c.Image, app.Spec.Pods[0].Image)
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

	err = fetchWithDefaults(appnn, &s)

	if err != nil {
		t.Error(err)
		return
	}

	// Simple test for service right expects there only to be the metrics port
	if len(s.Spec.Ports) > 1 {
		t.Errorf("Bad port count %d; expected 1", len(s.Spec.Ports))
	}

	if s.Spec.Ports[0].Port != env.Spec.Metrics.Port {
		t.Errorf("Bad port created %d; expected %d", s.Spec.Ports[0].Port, env.Spec.Metrics.Port)
	}

	// Kafka validation

	topic := strimzi.KafkaTopic{}
	topicName := "inventory-test-default"
	topicNamespacedName := types.NamespacedName{
		Namespace: env.Spec.Kafka.Namespace,
		Name:      topicName,
	}

	err = fetchWithDefaults(topicNamespacedName, &topic)

	if err != nil {
		t.Error(err)
		return
	}

	if *topic.Spec.Replicas != int32(32) {
		t.Errorf("Bad topic replica count %d; expected %d", *topic.Spec.Replicas, int32(32))
	}
	if *topic.Spec.Partitions != int32(5) {
		t.Errorf("Bad topic replica count %d; expected %d", *topic.Spec.Partitions, int32(5))
	}

	jsonContent, err := fetchConfig(name)

	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("%+v\n", jsonContent)

	cwConfigVals := map[string]string{
		"aws_access_key_id":     jsonContent.Logging.Cloudwatch.AccessKeyId,
		"aws_secret_access_key": jsonContent.Logging.Cloudwatch.SecretAccessKey,
		"log_group_name":        jsonContent.Logging.Cloudwatch.LogGroup,
		"aws_region":            jsonContent.Logging.Cloudwatch.Region,
	}

	for key, val := range cwData {
		if val != cwConfigVals[key] {
			t.Errorf("Wrong cloudwatch config value %s; expected %s", cwConfigVals[key], val)
			return
		}
	}

	if len(jsonContent.Kafka.Brokers[0].Hostname) == 0 {
		t.Error("Kafka broker hostname is not set")
		return
	}

	for i, kafkaTopic := range app.Spec.KafkaTopics {
		actual, expected := jsonContent.Kafka.Topics[i].RequestedName, kafkaTopic.TopicName

		if actual != expected {
			t.Errorf("Wrong topic name %s; expected %s", actual, expected)
		}
		actual, expected = jsonContent.Kafka.Topics[i].Name, topicName

		if actual != expected {
			t.Errorf("Wrong topic generated name %s; expected %s", actual, expected)
		}
	}
}

func TestConverterFuncs(t *testing.T) {
	answer, _ := utils.IntMin([]string{"4", "6", "7"})
	if answer != "4" {
		t.Errorf("Min function should have returned 4, returned %s", answer)
	}

	answer, _ = utils.IntMax([]string{"4", "6", "7"})
	if answer != "7" {
		t.Errorf("Min function should have returned 7, returned %s", answer)
	}

	answer, _ = utils.ListMerge([]string{"4,5,6", "6", "7,2"})
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
