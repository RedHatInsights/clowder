package controllers

import (
	"context"
	_ "embed"
	"os"

	cloudredhatcomv1alpha1 "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	cyndi "github.com/RedHatInsights/cyndi-operator/api/v1alpha1"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/prometheus/client_golang/prometheus"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	secretCompare schema.GroupVersionKind
	setupLog      = ctrl.Log.WithName("setup")
	Scheme        = runtime.NewScheme()
	CacheConfig   *rc.CacheConfig
	DebugOptions  rc.DebugOptions
	ProtectedGVKs = make(map[schema.GroupVersionKind]bool)
)

func init() {
	secretCompare, _ = utils.GetKindFromObj(Scheme, &core.Secret{})
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(cloudredhatcomv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(strimzi.AddToScheme(Scheme))
	utilruntime.Must(cyndi.AddToScheme(Scheme))
	utilruntime.Must(keda.AddToScheme(Scheme))
	utilruntime.Must(prom.AddToScheme(Scheme))
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

//go:embed version.txt
var Version string

// Run inits the manager and controllers and then starts the manager
func Run(metricsAddr string, probeAddr string, enableLeaderElection bool, config *rest.Config, signalHandler context.Context, enableWebHooks bool) {
	setupLog.Info("Loaded config", "config", clowderconfig.LoadedConfig)

	clowderVersion.With(prometheus.Labels{"version": Version}).Inc()

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 Scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "068b0003.cloud.redhat.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&ClowdAppReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClowdApp"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClowdApp")
		os.Exit(1)
	}
	if err = (&ClowdEnvironmentReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClowdEnvironment"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClowdEnvironment")
		os.Exit(1)
	}
	if err = (&ClowdJobInvocationReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClowdJobInvocation"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClowdJobInvocation")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if enableWebHooks {
		if err = (&crd.ClowdApp{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Captain")
			os.Exit(1)
		}
		mgr.GetWebhookServer().Register(
			"/mutate-pod",
			&webhook.Admission{
				Handler: &mutantPod{
					Client:   mgr.GetClient(),
					Recorder: mgr.GetEventRecorderFor("app"),
				},
			},
		)

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
