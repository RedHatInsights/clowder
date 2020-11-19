package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"

	cloudredhatcomv1alpha1 "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	strimzi "cloud.redhat.com/clowder/v2/apis/kafka.strimzi.io/v1beta1"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(cloudredhatcomv1alpha1.AddToScheme(scheme))
	utilruntime.Must(strimzi.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// Run inits the manager and controllers and then starts the manager
func Run(metricsAddr string, enableLeaderElection bool, config *rest.Config, signalHandler <-chan struct{}) {
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
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(signalHandler); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
	setupLog.Info("Exiting manager")
}

type ProxyClient struct {
	ResourceTracker *ResourceTracker
	pClient         client.Client
	Log             logr.Logger
}

type ResourceTracker struct {
	data map[string]map[string]bool
}

func (r *ResourceTracker) AddResource(obj runtime.Object) {
	if r.data == nil {
		r.data = make(map[string]map[string]bool)
	}

	var kind string

	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		kind = reflect.TypeOf(obj).String()
	} else {
		kind = obj.GetObjectKind().GroupVersionKind().Kind
	}

	var name string
	var rKind string

	switch kind {
	case "Deployment", "*v1.Deployment":
		rKind = "Deployment"
		dobj := obj.(*apps.Deployment)
		name = dobj.Name
	case "Service", "*v1.Service":
		rKind = "Service"
		dobj := obj.(*core.Service)
		name = dobj.Name
	case "PersistentVolumeClaim", "*v1.PersistentVolumeClaim":
		rKind = "PersistentVolumeClaim"
		dobj := obj.(*core.PersistentVolumeClaim)
		name = dobj.Name
	case "Secret", "*v1.Secret":
		rKind = "Secret"
		dobj := obj.(*core.Secret)
		name = dobj.Name
	default:
		fmt.Printf("DIDN:T DO IT SORRY %v, %v\n\n", kind, obj)
	}

	if _, ok := r.data[kind]; ok != true {
		r.data[rKind] = map[string]bool{}
	}

	r.data[rKind][name] = true
}

func (r *ResourceTracker) Reconcile(uid types.UID, client client.Client, ctx context.Context) {
	for k := range r.data {
		compareRef := func(name string, kind string, obj runtime.Object) {
			meta := obj.(metav1.Object)
			for _, ownerRef := range meta.GetOwnerReferences() {
				if ownerRef.UID == uid {
					if _, ok := r.data[kind][name]; ok != true {
						client.Delete(ctx, obj)
					}
				}
			}
		}

		switch k {
		case "Deployment", "*v1.Deployment":
			kind := "Deployment"
			objList := &apps.DeploymentList{}
			err := client.List(ctx, objList)
			fmt.Printf("%v", err)
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "Service", "*v1.Service":
			kind := "Service"
			objList := &core.ServiceList{}
			err := client.List(ctx, objList)
			fmt.Printf("%v", err)
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "PersistentVolumeClaim", "*v1.PersistentVolumeClaim":
			kind := "PersistentVolumeClaim"
			objList := &core.PersistentVolumeClaimList{}
			err := client.List(ctx, objList)
			fmt.Printf("%v", err)
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "Secret", "*v1.Secret":
			kind := "Secret"
			objList := &core.SecretList{}
			err := client.List(ctx, objList)
			fmt.Printf("%v", err)
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		}
	}
}

func (p ProxyClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return p.pClient.Get(ctx, key, obj)
}

func (p ProxyClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return p.pClient.List(ctx, list, opts...)
}

func (p ProxyClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	p.ResourceTracker.AddResource(obj)
	return p.pClient.Create(ctx, obj, opts...)
}

func (p ProxyClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	return p.pClient.Delete(ctx, obj, opts...)
}

func (p ProxyClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	p.ResourceTracker.AddResource(obj)
	return p.pClient.Update(ctx, obj, opts...)
}

func (p ProxyClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	p.ResourceTracker.AddResource(obj)
	return p.pClient.Patch(ctx, obj, patch, opts...)
}

func (p ProxyClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return p.pClient.DeleteAllOf(ctx, obj, opts...)
}

func (p ProxyClient) Status() client.StatusWriter {
	return p.Status()
}
