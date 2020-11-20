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

type ProxyClient struct {
	ResourceTracker map[string]map[string]bool
	Log             logr.Logger
	Ctx             context.Context
	client.Client
}

func (p ProxyClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	p.AddResource(obj)
	return p.Create(ctx, obj, opts...)
}

func (p ProxyClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	p.AddResource(obj)
	return p.Update(ctx, obj, opts...)
}

func (p ProxyClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	p.AddResource(obj)
	return p.Patch(ctx, obj, patch, opts...)
}

func (p *ProxyClient) AddResource(obj runtime.Object) {
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

	if _, ok := p.ResourceTracker[kind]; ok != true {
		p.ResourceTracker[rKind] = map[string]bool{}
	}

	p.ResourceTracker[rKind][name] = true
}

func (p *ProxyClient) Reconcile(uid types.UID) error {
	for k := range p.ResourceTracker {
		compareRef := func(name string, kind string, obj runtime.Object) error {
			meta := obj.(metav1.Object)
			for _, ownerRef := range meta.GetOwnerReferences() {
				if ownerRef.UID == uid {
					if _, ok := p.ResourceTracker[kind][name]; ok != true {
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
		case "Deployment", "*v1.Deployment":
			kind := "Deployment"
			objList := &apps.DeploymentList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "Service", "*v1.Service":
			kind := "Service"
			objList := &core.ServiceList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "PersistentVolumeClaim", "*v1.PersistentVolumeClaim":
			kind := "PersistentVolumeClaim"
			objList := &core.PersistentVolumeClaimList{}
			err := p.List(p.Ctx, objList)
			if err != nil {
				return err
			}
			for _, obj := range objList.Items {
				compareRef(obj.Name, kind, &obj)
			}
		case "Secret", "*v1.Secret":
			kind := "Secret"
			objList := &core.SecretList{}
			err := p.List(p.Ctx, objList)
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
