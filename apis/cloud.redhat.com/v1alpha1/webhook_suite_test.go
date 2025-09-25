/*
Copyright 2021.

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

package v1alpha1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	g.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(t,
		"Webhook Suite",
	)
}

var _ = ginkgo.BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	ginkgo.By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "config", "webhook")},
		},
	}

	cfg, err := testEnv.Start()
	g.Expect(err).NotTo(g.HaveOccurred())
	g.Expect(cfg).NotTo(g.BeNil())

	scheme := runtime.NewScheme()
	err = AddToScheme(scheme)
	g.Expect(err).NotTo(g.HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme)
	g.Expect(err).NotTo(g.HaveOccurred())

	err = v1.AddToScheme(scheme)
	g.Expect(err).NotTo(g.HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	g.Expect(err).NotTo(g.HaveOccurred())
	g.Expect(k8sClient).NotTo(g.BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
	})
	g.Expect(err).NotTo(g.HaveOccurred())

	err = (&ClowdApp{}).SetupWebhookWithManager(mgr)
	g.Expect(err).NotTo(g.HaveOccurred())

	//+kubebuilder:scaffold:webhook

	go func() {
		err = mgr.Start(ctx)
		if err != nil {
			g.Expect(err).NotTo(g.HaveOccurred())
		}
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	g.Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
		if err != nil {
			return err
		}
		err = conn.Close()
		if err != nil {
			return err
		}
		return nil
	}).Should(g.Succeed())

})

var _ = ginkgo.AfterSuite(func() {
	cancel()
	ginkgo.By("tearing down the test environment")
	err := testEnv.Stop()
	g.Expect(err).NotTo(g.HaveOccurred())
})

var _ = ginkgo.Describe("ClowdApp webhook", func() {
	ginkgo.Context("When creating ClowdApp", func() {
		var testNamespace1 *v1.Namespace
		var testNamespace2 *v1.Namespace
		var testEnv1 *ClowdEnvironment
		var testEnv2 *ClowdEnvironment

		ginkgo.BeforeEach(func() {
			// Create two test namespaces for cross-namespace testing
			testNamespace1 = &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-webhook-1-" + fmt.Sprintf("%d", time.Now().UnixNano()),
				},
			}
			err := k8sClient.Create(ctx, testNamespace1)
			g.Expect(err).NotTo(g.HaveOccurred())

			testNamespace2 = &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-webhook-2-" + fmt.Sprintf("%d", time.Now().UnixNano()),
				},
			}
			err = k8sClient.Create(ctx, testNamespace2)
			g.Expect(err).NotTo(g.HaveOccurred())

			// Create test environments in both namespaces
			testEnv1 = &ClowdEnvironment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-env-1",
					Namespace: testNamespace1.Name,
				},
				Spec: ClowdEnvironmentSpec{
					TargetNamespace: testNamespace1.Name,
					Providers: ProvidersConfig{
						Web: WebConfig{
							Port: 8000,
							Mode: "operator",
						},
						Metrics: MetricsConfig{
							Port: 9000,
							Mode: "operator",
							Path: "/metrics",
						},
						Kafka: KafkaConfig{
							Mode: "none",
						},
						Database: DatabaseConfig{
							Mode: "none",
						},
						Logging: LoggingConfig{
							Mode: "none",
						},
						ObjectStore: ObjectStoreConfig{
							Mode: "none",
						},
						InMemoryDB: InMemoryDBConfig{
							Mode: "none",
						},
					},
				},
			}
			err = k8sClient.Create(ctx, testEnv1)
			g.Expect(err).NotTo(g.HaveOccurred())

			testEnv2 = &ClowdEnvironment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-env-2",
					Namespace: testNamespace2.Name,
				},
				Spec: ClowdEnvironmentSpec{
					TargetNamespace: testNamespace2.Name,
					Providers: ProvidersConfig{
						Web: WebConfig{
							Port: 8000,
							Mode: "operator",
						},
						Metrics: MetricsConfig{
							Port: 9000,
							Mode: "operator",
							Path: "/metrics",
						},
						Kafka: KafkaConfig{
							Mode: "none",
						},
						Database: DatabaseConfig{
							Mode: "none",
						},
						Logging: LoggingConfig{
							Mode: "none",
						},
						ObjectStore: ObjectStoreConfig{
							Mode: "none",
						},
						InMemoryDB: InMemoryDBConfig{
							Mode: "none",
						},
					},
				},
			}
			err = k8sClient.Create(ctx, testEnv2)
			g.Expect(err).NotTo(g.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			// Clean up test resources
			err := k8sClient.Delete(ctx, testEnv1)
			g.Expect(err).NotTo(g.HaveOccurred())
			err = k8sClient.Delete(ctx, testEnv2)
			g.Expect(err).NotTo(g.HaveOccurred())
			err = k8sClient.Delete(ctx, testNamespace1)
			g.Expect(err).NotTo(g.HaveOccurred())
			err = k8sClient.Delete(ctx, testNamespace2)
			g.Expect(err).NotTo(g.HaveOccurred())
		})

		ginkgo.It("should allow creating a ClowdApp with unique name", func() {
			clowdApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unique-app",
					Namespace: testNamespace1.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-1",
					Deployments: []Deployment{
						{
							Name: "processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, clowdApp)
			g.Expect(err).NotTo(g.HaveOccurred())
		})

		ginkgo.It("should allow creating ClowdApps with same name in different ClowdEnvironments", func() {
			// First, create a ClowdApp in namespace 1 with env-1
			firstApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "same-name-app",
					Namespace: testNamespace1.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-1",
					Deployments: []Deployment{
						{
							Name: "processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, firstApp)
			g.Expect(err).NotTo(g.HaveOccurred())

			// Now create another ClowdApp with the same name in namespace 2 with env-2
			secondApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "same-name-app",
					Namespace: testNamespace2.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-2",
					Deployments: []Deployment{
						{
							Name: "different-processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
							},
						},
					},
				},
			}

			err = k8sClient.Create(ctx, secondApp)
			g.Expect(err).NotTo(g.HaveOccurred())
		})

		ginkgo.It("should reject creating ClowdApps with duplicate name in same ClowdEnvironment", func() {
			// First, create a ClowdApp in namespace 1 with env-1
			firstApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-app",
					Namespace: testNamespace1.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-1",
					Deployments: []Deployment{
						{
							Name: "processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, firstApp)
			g.Expect(err).NotTo(g.HaveOccurred())

			// Now try to create another ClowdApp with the same name in namespace 2 but same env
			secondApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-app",
					Namespace: testNamespace2.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-1", // Same environment as first app
					Deployments: []Deployment{
						{
							Name: "different-processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
							},
						},
					},
				},
			}

			err = k8sClient.Create(ctx, secondApp)
			g.Expect(err).To(g.HaveOccurred())
			g.Expect(errors.IsInvalid(err)).To(g.BeTrue())
			g.Expect(err.Error()).To(g.ContainSubstring("already exists"))
		})

		ginkgo.It("should allow updating a ClowdApp without changing name", func() {
			// Create a ClowdApp
			clowdApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "update-app",
					Namespace: testNamespace1.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-1",
					Deployments: []Deployment{
						{
							Name: "processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, clowdApp)
			g.Expect(err).NotTo(g.HaveOccurred())

			// Update the ClowdApp (change image)
			clowdApp.Spec.Deployments[0].PodSpec.Image = "quay.io/psav/clowder-hello:v2"
			err = k8sClient.Update(ctx, clowdApp)
			g.Expect(err).NotTo(g.HaveOccurred())
		})

		ginkgo.It("should validate database configuration", func() {
			// Try to create a ClowdApp with invalid database configuration
			invalidApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-db-app",
					Namespace: testNamespace1.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-1",
					Database: DatabaseSpec{
						Name:            "my-db",
						SharedDBAppName: "shared-db-app",
					},
					Deployments: []Deployment{
						{
							Name: "processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, invalidApp)
			g.Expect(err).To(g.HaveOccurred())
			g.Expect(errors.IsInvalid(err)).To(g.BeTrue())
			g.Expect(err.Error()).To(g.ContainSubstring("cannot set db name and sharedDbApp Name together"))
		})

		ginkgo.It("should validate sidecar configuration", func() {
			// Try to create a ClowdApp with invalid sidecar configuration
			invalidApp := &ClowdApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-sidecar-app",
					Namespace: testNamespace1.Name,
				},
				Spec: ClowdAppSpec{
					EnvName: "test-env-1",
					Deployments: []Deployment{
						{
							Name: "processor",
							PodSpec: PodSpec{
								Image: "quay.io/psav/clowder-hello",
								Sidecars: []Sidecar{
									{
										Name:    "invalid-sidecar",
										Enabled: true,
									},
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, invalidApp)
			g.Expect(err).To(g.HaveOccurred())
			g.Expect(errors.IsInvalid(err)).To(g.BeTrue())
			g.Expect(err.Error()).To(g.ContainSubstring("Sidecar is of unknown type"))
		})
	})
})
