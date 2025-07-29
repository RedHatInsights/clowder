package controllers

import (
	"context"
	_ "embed"
	"os"

	cyndi "github.com/RedHatInsights/cyndi-operator/api/v1alpha1"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	cert "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	keda "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	sub "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/metrics/subscriptions"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	secretCompare schema.GroupVersionKind
	setupLog      = ctrl.Log.WithName("setup")
	// Scheme defines the runtime scheme for the Clowder controller
	Scheme = runtime.NewScheme()
	// CacheConfig holds the cache configuration for the Clowder controller
	CacheConfig *rc.CacheConfig
	// DebugOptions holds the debug configuration options for the Clowder controller
	DebugOptions rc.DebugOptions
	// ProtectedGVKs holds the map of protected GroupVersionKinds that should not be deleted
	ProtectedGVKs = make(map[schema.GroupVersionKind]bool)
)

func init() {
	secretCompare, _ = utils.GetKindFromObj(Scheme, &core.Secret{})
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(crd.AddToScheme(Scheme))
	utilruntime.Must(strimzi.AddToScheme(Scheme))
	utilruntime.Must(cyndi.AddToScheme(Scheme))
	utilruntime.Must(keda.AddToScheme(Scheme))
	utilruntime.Must(prom.AddToScheme(Scheme))
	utilruntime.Must(sub.AddToScheme(Scheme))
	utilruntime.Must(cert.AddToScheme(Scheme))
	// +kubebuilder:scaffold:scheme

	// Add certain resources so that they will be protected an not get deleted
	gvk, _ := utils.GetKindFromObj(Scheme, &strimzi.KafkaTopic{})
	ProtectedGVKs[gvk] = true

	if !clowderconfig.LoadedConfig.Features.KedaResources {
		gvk, _ := utils.GetKindFromObj(Scheme, &keda.ScaledObject{})
		ProtectedGVKs[gvk] = true
	}

	DebugOptions = rc.DebugOptions{
		Create: clowderconfig.LoadedConfig.DebugOptions.Cache.Create,
		Update: clowderconfig.LoadedConfig.DebugOptions.Cache.Update,
		Apply:  clowderconfig.LoadedConfig.DebugOptions.Cache.Apply,
	}
}

// Version contains the current version of Clowder
//
//go:embed version.txt
var Version string

func printConfig() error {
	setupLog.Info("Loaded config", "config", clowderconfig.LoadedConfig)
	return nil
}

// Run inits the manager and controllers and then starts the manager
func Run(signalHandler context.Context, metricsAddr string, probeAddr string, enableLeaderElection bool, config *rest.Config, enableWebHooks bool) {
	err := printConfig()
	if err != nil {
		setupLog.Error(err, "unable to print config")
		os.Exit(1)
	}

	clowderVersion.With(prometheus.Labels{"version": Version}).Inc()

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: Scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "068b0003.cloud.redhat.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	if err := addControllersToManager(mgr); err != nil {
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := setupWebhooks(mgr, enableWebHooks); err != nil {
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(signalHandler); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
	setupLog.Info("Exiting manager")
}

func addControllersToManager(mgr manager.Manager) error {
	AppHashCache := hashcache.NewHashCache()
	EnvHashCache := hashcache.NewHashCache()

	if err := (&ClowdAppReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("ClowdApp"),
		Scheme:    mgr.GetScheme(),
		HashCache: &AppHashCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClowdApp")
		return err
	}
	if err := (&ClowdEnvironmentReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("ClowdEnvironment"),
		Scheme:    mgr.GetScheme(),
		HashCache: &EnvHashCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClowdEnvironment")
		return err
	}
	if err := (&ClowdJobInvocationReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClowdJobInvocation"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClowdJobInvocation")
		return err
	}
	return nil
}

func setupWebhooks(mgr manager.Manager, enableWebHooks bool) error {
	if enableWebHooks {
		if err := (&crd.ClowdApp{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Captain")
			return err
		}
		mgr.GetWebhookServer().Register(
			"/mutate-pod",
			&webhook.Admission{
				Handler: &mutantPod{
					Decoder:  admission.NewDecoder(mgr.GetScheme()),
					Client:   mgr.GetClient(),
					Recorder: mgr.GetEventRecorderFor("app"),
				},
			},
		)
	}
	return nil
}
