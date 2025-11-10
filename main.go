// Package main provides the entry point for the Clowder controller manager
package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	// _ "net/http/pprof" // Commented out to avoid exposing profiling endpoint (gosec G108)

	"go.uber.org/zap"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/zapr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/RedHatInsights/rhc-osdk-utils/logging"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	controllers "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(crd.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func loggerSync(log *zap.Logger) {
	// Ignore the error from sync
	_ = log.Sync()
}

func runAPIServer() {
	server := http.Server{
		Addr:              "localhost:8000",
		ReadHeaderTimeout: 2 * time.Second,
	}
	// Ignore error from starting pprof
	_ = server.ListenAndServe()
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	// This metrics address may need to be 9443
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	logger, err := logging.SetupLoggingWithLevel(clowderconfig.LoadedConfig.Features.DisableCloudWatchLogging, -1)

	if err != nil {
		panic(err)
	}

	ctrl.SetLogger(zapr.NewLogger(logger))

	defer loggerSync(logger)

	if clowderconfig.LoadedConfig.DebugOptions.Pprof.Enable {
		go runAPIServer()
	}

	go func() {
		fmt.Println(controllers.CreateAPIServer().ListenAndServe())
	}()

	controllers.Run(ctrl.SetupSignalHandler(), metricsAddr, probeAddr, enableLeaderElection, ctrl.GetConfigOrDie(), !clowderconfig.LoadedConfig.Features.DisableWebhooks)
}
