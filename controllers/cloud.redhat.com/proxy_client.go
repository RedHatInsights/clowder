package controllers

import (
	"context"
	"reflect"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProxyClient is a type that holds mimicks a real client but tracks certain resources through
// creation, so that they can be deleted when a ClowdApp/ClowdEnv is deleted.
type ProxyClient struct {
	ResourceTracker map[string]map[string]bool
	Ctx             context.Context
	client.Client
}

// Get proxies the Get call to the real client, running through tracking first
func (p *ProxyClient) Get(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
	clientOpsMetric.With(prometheus.Labels{"operation": "get"}).Inc()
	return p.Client.Get(ctx, key, obj)
}

// Create proxies the Create call to the real client, running through tracking first
func (p *ProxyClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	p.AddResource(obj)
	clientOpsMetric.With(prometheus.Labels{"operation": "create"}).Inc()
	return p.Client.Create(ctx, obj, opts...)
}

// Update proxies the Update call to the real client, running through tracking first
func (p *ProxyClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	p.AddResource(obj)
	clientOpsMetric.With(prometheus.Labels{"operation": "update"}).Inc()
	return p.Client.Update(ctx, obj, opts...)
}

// Patch proxies the Patch call to the real client, running through tracking first
func (p *ProxyClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	p.AddResource(obj)
	clientOpsMetric.With(prometheus.Labels{"operation": "patch"}).Inc()
	return p.Client.Patch(ctx, obj, patch, opts...)
}

// Delete proxies the Delete call to the real client, running through tracking first
func (p *ProxyClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	clientOpsMetric.With(prometheus.Labels{"operation": "delete"}).Inc()
	return p.Client.Delete(ctx, obj, opts...)
}

// AddResource adds the given resource to the tracking database so that it can be checked for existence
// after the deletion of a ClowdObject.
func (p *ProxyClient) AddResource(obj runtime.Object) {
	log := (*p.Ctx.Value(errors.ClowdKey("log")).(*logr.Logger)).WithName("proxy-client")

	if p.ResourceTracker == nil {
		p.ResourceTracker = make(map[string]map[string]bool)
	}

	var kind string

	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		kind = reflect.TypeOf(obj).String()
	} else {
		kind = obj.GetObjectKind().GroupVersionKind().Kind
	}

	var name string
	var namespace string
	var rKind string

	switch kind {
	case "ConfigMap", "*v1.ConfigMap":
		rKind = "ConfigMap"
		dobj := obj.(*core.ConfigMap)
		name = dobj.Name
		namespace = dobj.Namespace
	case "Deployment", "*v1.Deployment":
		rKind = "Deployment"
		dobj := obj.(*apps.Deployment)
		name = dobj.Name
		namespace = dobj.Namespace
	case "Service", "*v1.Service":
		rKind = "Service"
		dobj := obj.(*core.Service)
		name = dobj.Name
		namespace = dobj.Namespace
	case "PersistentVolumeClaim", "*v1.PersistentVolumeClaim":
		rKind = "PersistentVolumeClaim"
		dobj := obj.(*core.PersistentVolumeClaim)
		name = dobj.Name
		namespace = dobj.Namespace
	case "Secret", "*v1.Secret":
		rKind = "Secret"
		dobj := obj.(*core.Secret)
		name = dobj.Name
		namespace = dobj.Namespace
	case "CronJob", "*v1beta1.CronJob":
		rKind = "CronJob"
		dobj := obj.(*batch.CronJob)
		name = dobj.Name
		namespace = dobj.Namespace
	default:
		return
	}

	if _, ok := p.ResourceTracker[rKind]; ok != true {
		p.ResourceTracker[rKind] = map[string]bool{}
	}

	log.Info("Tracking resource", "kind", rKind, "namespace", namespace, "name", name)
	p.ResourceTracker[rKind][name] = true
}

var gvkMap = map[string]schema.GroupVersionKind{
	"ConfigMap":             {Group: "", Version: "v1", Kind: "ConfigMap"},
	"Deployment":            {Group: "apps", Version: "v1", Kind: "Deployment"},
	"Service":               {Group: "", Version: "v1", Kind: "Service"},
	"PersistentVolumeClaim": {Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
	"Secret":                {Group: "", Version: "v1", Kind: "Secret"},
	"CronJob":               {Group: "batch", Version: "v1beta1", Kind: "CronJob"},
}

// Reconcile goes through all resources in the resource tracker that still exist after a ClowdObject
// is deleted and removes those objects. Since Clowder created them, they are assumed safe to delete.
func (p *ProxyClient) Reconcile(uid types.UID) error {
	log := (*p.Ctx.Value(errors.ClowdKey("log")).(*logr.Logger)).WithName("proxy-client")
	for k := range p.ResourceTracker {

		gvk := gvkMap[k]
		nobjList := unstructured.UnstructuredList{}
		nobjList.SetGroupVersionKind(gvk)

		err := p.List(p.Ctx, &nobjList)
		if err != nil {
			return err
		}
		for _, obj := range nobjList.Items {
			for _, ownerRef := range obj.GetOwnerReferences() {
				if ownerRef.UID == uid {
					if _, ok := p.ResourceTracker[gvk.Kind][obj.GetName()]; ok != true {
						log.Info("Deleting resource", "kind", gvk.Kind, "name", obj.GetName())
						err := p.Delete(p.Ctx, &obj)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}
