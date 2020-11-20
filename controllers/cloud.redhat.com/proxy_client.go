package controllers

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
		return
	}

	if _, ok := r.data[kind]; ok != true {
		r.data[rKind] = map[string]bool{}
	}

	r.data[rKind][name] = true
}

func (r *ResourceTracker) Reconcile(uid types.UID, client client.Client, ctx context.Context) error {
	for k := range r.data {
		compareRef := func(name string, kind string, obj runtime.Object) error {
			meta := obj.(metav1.Object)
			for _, ownerRef := range meta.GetOwnerReferences() {
				if ownerRef.UID == uid {
					if _, ok := r.data[kind][name]; ok != true {
						err := client.Delete(ctx, obj)
						if err != nil {
							return err
						}
					}
				}
			}
			return nil
		}

		switch k {
		case "Deployment", "*v1.Deployment":
			kind := "Deployment"
			objList := &apps.DeploymentList{}
			err := client.List(ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "Service", "*v1.Service":
			kind := "Service"
			objList := &core.ServiceList{}
			err := client.List(ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "PersistentVolumeClaim", "*v1.PersistentVolumeClaim":
			kind := "PersistentVolumeClaim"
			objList := &core.PersistentVolumeClaimList{}
			err := client.List(ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "Secret", "*v1.Secret":
			kind := "Secret"
			objList := &core.SecretList{}
			err := client.List(ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		}
	}
	return nil
}

type ProxyClient struct {
	ResourceTracker *ResourceTracker
	Log             logr.Logger
	client.Client
}

func (p ProxyClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	p.ResourceTracker.AddResource(obj)
	return p.Create(ctx, obj, opts...)
}

func (p ProxyClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	p.ResourceTracker.AddResource(obj)
	return p.Update(ctx, obj, opts...)
}

func (p ProxyClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	p.ResourceTracker.AddResource(obj)
	return p.Patch(ctx, obj, patch, opts...)
}
