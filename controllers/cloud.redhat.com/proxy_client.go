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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProxyClient struct {
	ResourceTracker map[string]map[string]bool
	Ctx             context.Context
	client.Client
}

func (p *ProxyClient) Get(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
	clientOpsMetric.With(prometheus.Labels{"operation": "get"}).Inc()
	return p.Client.Get(ctx, key, obj)
}

func (p *ProxyClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	p.AddResource(obj)
	clientOpsMetric.With(prometheus.Labels{"operation": "create"}).Inc()
	return p.Client.Create(ctx, obj, opts...)
}

func (p *ProxyClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	p.AddResource(obj)
	clientOpsMetric.With(prometheus.Labels{"operation": "update"}).Inc()
	return p.Client.Update(ctx, obj, opts...)
}

func (p *ProxyClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	p.AddResource(obj)
	clientOpsMetric.With(prometheus.Labels{"operation": "patch"}).Inc()
	return p.Client.Patch(ctx, obj, patch, opts...)
}

func (p *ProxyClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	clientOpsMetric.With(prometheus.Labels{"operation": "delete"}).Inc()
	return p.Client.Delete(ctx, obj, opts...)
}

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

func (p *ProxyClient) Reconcile(uid types.UID) error {
	log := (*p.Ctx.Value(errors.ClowdKey("log")).(*logr.Logger)).WithName("proxy-client")
	for k := range p.ResourceTracker {
		compareRef := func(name string, kind string, obj runtime.Object) error {
			meta := obj.(metav1.Object)
			for _, ownerRef := range meta.GetOwnerReferences() {
				if ownerRef.UID == uid {
					if _, ok := p.ResourceTracker[kind][name]; ok != true {
						kind := obj.GetObjectKind().GroupVersionKind().Kind
						name := meta.GetName()
						log.Info("Deleting resource", "kind", kind, "name", name)
						err := p.Delete(p.Ctx, obj)
						if err != nil {
							return err
						}
					}
				}
			}
			return nil
		}

		switch k {
		case "ConfigMap", "*v1.ConfigMap":
			kind := "ConfigMap"
			objList := &core.ConfigMapList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				err := compareRef(obj.Name, kind, &obj)
				if err != nil {
					return err
				}
			}
		case "Deployment", "*v1.Deployment":
			kind := "Deployment"
			objList := &apps.DeploymentList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				err := compareRef(obj.Name, kind, &obj)
				if err != nil {
					return err
				}
			}
		case "Service", "*v1.Service":
			kind := "Service"
			objList := &core.ServiceList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				err := compareRef(obj.Name, kind, &obj)
				if err != nil {
					return err
				}
			}
		case "PersistentVolumeClaim", "*v1.PersistentVolumeClaim":
			kind := "PersistentVolumeClaim"
			objList := &core.PersistentVolumeClaimList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				err := compareRef(obj.Name, kind, &obj)
				if err != nil {
					return err
				}
			}
		case "Secret", "*v1.Secret":
			kind := "Secret"
			objList := &core.SecretList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				err := compareRef(obj.Name, kind, &obj)
				if err != nil {
					return err
				}
			}
		case "CronJob", "*v1beta1.CronJob":
			kind := "CronJob"
			objList := &batch.CronJobList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				err := compareRef(obj.Name, kind, &obj)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
