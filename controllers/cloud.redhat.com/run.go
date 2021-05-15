package controllers

import (
	"os"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/clowder_config"

	cloudredhatcomv1alpha1 "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	cyndi "cloud.redhat.com/clowder/v2/apis/cyndi-operator/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta1"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type gvks struct {
	List   schema.GroupVersionKind
	Single schema.GroupVersionKind
}

var (
	scheme      = runtime.NewScheme()
	setupLog    = ctrl.Log.WithName("setup")
	gvksForType = make(map[string]gvks)
)

var secretCompare schema.GroupVersionKind

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cloudredhatcomv1alpha1.AddToScheme(scheme))
	utilruntime.Must(strimzi.AddToScheme(scheme))
	utilruntime.Must(cyndi.AddToScheme(scheme))
	utilruntime.Must(prom.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	secretCompare, _ = utils.GetKindFromObj(scheme, &core.Secret{})

	// Get GVKs for types we are interested in tracking status of
	listGVK, _ := utils.GetKindFromObj(scheme, &apps.DeploymentList{})
	gvk, _ := utils.GetKindFromObj(scheme, &apps.Deployment{})
	gvksForType["deployment"] = gvks{List: listGVK, Single: gvk}

	listGVK, _ = utils.GetKindFromObj(scheme, &strimzi.KafkaList{})
	gvk, _ = utils.GetKindFromObj(scheme, &strimzi.Kafka{})
	gvksForType["kafka"] = gvks{List: listGVK, Single: gvk}

	listGVK, _ = utils.GetKindFromObj(scheme, &strimzi.KafkaConnectList{})
	gvk, _ = utils.GetKindFromObj(scheme, &strimzi.KafkaConnect{})
	gvksForType["kafkaconnect"] = gvks{List: listGVK, Single: gvk}
}

// Run inits the manager and controllers and then starts the manager
func Run(metricsAddr string, enableLeaderElection bool, config *rest.Config, signalHandler <-chan struct{}) {
	setupLog.Info("Loaded config", "config", clowder_config.LoadedConfig)

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "068b0003.cloud.redhat.com",
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

	setupLog.Info("starting manager")
	if err := mgr.Start(signalHandler); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
	setupLog.Info("Exiting manager")
}
