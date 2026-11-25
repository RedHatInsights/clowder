package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/kafka"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/logging"
)

func TestManagedKafkaSecretUpdateEnqueuesApp(t *testing.T) {
	for _, secretNamespace := range []string{"workload", "shared-config"} {
		t.Run(secretNamespace, func(t *testing.T) {
			ctx := context.Background()
			secret := &core.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "managed-secret", Namespace: secretNamespace},
				Data: map[string][]byte{
					"hostname": []byte("broker.example"),
					"port":     []byte("9092"),
					"username": []byte("test-user"),
					"password": []byte("initial"),
				},
			}
			app := &crd.ClowdApp{
				ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "workload"},
				Spec: crd.ClowdAppSpec{
					EnvName:     "environment",
					KafkaTopics: []crd.KafkaTopicSpec{{TopicName: "events"}},
				},
			}
			env := &crd.ClowdEnvironment{
				ObjectMeta: metav1.ObjectMeta{Name: app.Spec.EnvName},
				Spec: crd.ClowdEnvironmentSpec{Providers: crd.ProvidersConfig{
					Kafka: crd.KafkaConfig{Mode: "managed", ManagedSecretRef: crd.NamespacedName{
						Name: secret.Name, Namespace: secret.Namespace,
					}},
				}},
			}
			cachedClient := fake.NewClientBuilder().WithScheme(Scheme).WithObjects(secret, app).Build()
			cache := hashcache.NewHashCache()
			provider, err := kafka.NewManagedKafka(&providers.Provider{
				Ctx: ctx, Client: cachedClient, Env: env, HashCache: &cache, Config: &config.AppConfig{},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := provider.Provide(app); err != nil {
				t.Fatal(err)
			}
			updated := secret.DeepCopy()
			updated.Data["password"] = []byte("rotated")
			queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
			defer queue.ShutDown()
			handler := &enqueueRequestForObjectCustom{
				TypeOfOwner: &crd.ClowdApp{}, hashCache: &cache, client: cachedClient, logr: logr.Discard(),
			}
			handler.Update(ctx, event.UpdateEvent{ObjectOld: secret, ObjectNew: updated}, queue)

			if queue.Len() != 1 {
				t.Fatalf("managed Kafka secret update must enqueue its consuming ClowdApp; got %d requests", queue.Len())
			}
			request, _ := queue.Get()
			defer queue.Done(request)
			want := types.NamespacedName{Name: app.Name, Namespace: app.Namespace}
			if request.NamespacedName != want {
				t.Fatalf("queued %v, want consuming app %v", request.NamespacedName, want)
			}
		})
	}
}

func TestCloudwatchSecretUpdateEnqueuesApp(t *testing.T) {
	ctx := context.Background()
	secret := &core.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "cloudwatch", Namespace: "workload"},
		Data: map[string][]byte{
			"aws_access_key_id":     []byte("access-key-id"),
			"aws_secret_access_key": []byte("initial"),
			"aws_region":            []byte("eu-central-1"),
			"log_group_name":        []byte("Logs_Stage"),
		},
	}
	app := &crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "workload"},
		Spec:       crd.ClowdAppSpec{EnvName: "environment"},
	}
	env := &crd.ClowdEnvironment{ObjectMeta: metav1.ObjectMeta{Name: app.Spec.EnvName}}
	cachedClient := fake.NewClientBuilder().WithScheme(Scheme).WithObjects(secret, app).Build()
	cache := hashcache.NewHashCache()
	provider := logging.NewAppInterfaceLogging(&providers.Provider{
		Ctx: ctx, Client: cachedClient, Env: env, HashCache: &cache, Config: &config.AppConfig{},
	})
	if err := provider.Provide(app); err != nil {
		t.Fatal(err)
	}
	updated := secret.DeepCopy()
	updated.Data["aws_secret_access_key"] = []byte("rotated")
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
	defer queue.ShutDown()
	handler := &enqueueRequestForObjectCustom{
		TypeOfOwner: &crd.ClowdApp{}, hashCache: &cache, client: cachedClient, logr: logr.Discard(),
	}
	handler.Update(ctx, event.UpdateEvent{ObjectOld: secret, ObjectNew: updated}, queue)

	if queue.Len() != 1 {
		t.Fatalf("cloudwatch secret update must enqueue its consuming ClowdApp; got %d requests", queue.Len())
	}
	request, _ := queue.Get()
	defer queue.Done(request)
	want := types.NamespacedName{Name: app.Name, Namespace: app.Namespace}
	if request.NamespacedName != want {
		t.Fatalf("queued %v, want consuming app %v", request.NamespacedName, want)
	}
}
