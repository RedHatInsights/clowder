package controllers

import (
	"context"
	"fmt"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/hashcache"
)

type enqueueRequestForObjectCustom struct {
	HandlerFuncs HandlerFuncs
	groupKind    schema.GroupKind
	TypeOfOwner  runtime.Object
	mapObj       meta.RESTMapper
	logr         logr.Logger
	ctrlName     string
	client       client.Client
	hashCache    *hashcache.HashCache
}

var _ handler.EventHandler = &enqueueRequestForObjectCustom{}

type HandlerFuncBuilder func(logr logr.Logger, ctrlName string) HandlerFuncs

func createNewHandler(mgr manager.Manager, scheme *runtime.Scheme, input HandlerFuncBuilder, log logr.Logger, ctrlName string, typeOfOwner runtime.Object, hashCache *hashcache.HashCache) (handler.EventHandler, error) {
	handleFuncs := input(log, ctrlName)

	kinds, _, err := scheme.ObjectKinds(typeOfOwner)
	if err != nil {
		return nil, err
	}

	groupKind := schema.GroupKind{Group: kinds[0].Group, Kind: kinds[0].Kind}

	obj := enqueueRequestForObjectCustom{
		HandlerFuncs: handleFuncs,
		TypeOfOwner:  typeOfOwner,
		logr:         log,
		ctrlName:     ctrlName,
		hashCache:    hashCache,
		client:       mgr.GetClient(),
		groupKind:    groupKind,
		mapObj:       mgr.GetRESTMapper(),
	}
	return &obj, nil
}

func (e *enqueueRequestForObjectCustom) findOwner(a client.Object) (*types.NamespacedName, string) {
	ownref := metav1.GetControllerOf(a)
	if ownref == nil {
		return nil, ""
	}

	refGVK, err := schema.ParseGroupVersion(ownref.APIVersion)
	if err != nil {
		return nil, ""
	}

	if ownref.Kind == e.groupKind.Kind && refGVK.Group == e.groupKind.Group {
		nn := types.NamespacedName{
			Name: ownref.Name,
		}
		mapping, err := e.mapObj.RESTMapping(e.groupKind, refGVK.Version)
		if err != nil {
			return nil, ""
		}
		if mapping.Scope.Name() != meta.RESTScopeNameRoot {
			nn.Namespace = a.GetNamespace()
		}
		return &nn, ownref.Kind
	}
	return nil, ""
}

func (e *enqueueRequestForObjectCustom) getOwner(obj client.Object) (*types.NamespacedName, string) {
	if obj == nil {
		return nil, ""
	}
	return e.findOwner(obj)
}

func (e *enqueueRequestForObjectCustom) logMessage(obj client.Object, msg string, toKind string, own *types.NamespacedName) {
	gvk, _ := utils.GetKindFromObj(Scheme, obj)
	logMessage(e.logr, "Reconciliation trigger", "ctrl", e.ctrlName, "type", msg, "resType", gvk.Kind, "sourceObj", fmt.Sprintf("%s/%s/%s", gvk.Kind, obj.GetNamespace(), obj.GetName()), "destObj", fmt.Sprintf("%s/%s/%s", toKind, own.Namespace, own.Name))
}

func (e *enqueueRequestForObjectCustom) updateHashCacheForConfigMapAndSecret(obj client.Object) (bool, error) {
	switch obj.(type) {
	case *core.ConfigMap, *core.Secret:
		if obj.GetAnnotations()[clowderconfig.LoadedConfig.Settings.RestarterAnnotationName] == "true" {
			return e.hashCache.CreateOrUpdateObject(obj, false)
		}
		hcOjb, err := e.hashCache.Read(obj)
		if err != nil {
			return false, err
		}
		if hcOjb.Always {
			return e.hashCache.CreateOrUpdateObject(obj, false)
		}
	}
	return false, nil
}

func getNamespacedName(obj client.Object) *types.NamespacedName {
	return &types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

func (e *enqueueRequestForObjectCustom) Create(ctx context.Context, evt event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	shouldUpdate, err := e.updateHashCacheForConfigMapAndSecret(evt.Object)
	if err != nil {
		e.logMessage(evt.Object, err.Error(), "", getNamespacedName(evt.Object))
	}

	if shouldUpdate {
		_ = e.doUpdateToHash(evt.Object, q)
		e.reconcileAllAppsUsingObject(ctx, evt.Object, q)
	}

	if own, toKind := e.getOwner(evt.Object); own != nil {
		if doRequest, msg := e.HandlerFuncs.CreateFunc(evt); doRequest {
			e.logMessage(evt.Object, msg, toKind, own)
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}

func (e *enqueueRequestForObjectCustom) doUpdateToHash(obj client.Object, q workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	e.logMessage(obj, "update needed because changed", "", getNamespacedName(obj))
	hashObj, err := e.hashCache.Read(obj)
	if err != nil {
		e.logMessage(obj, err.Error(), "", getNamespacedName(obj))
		return err
	}

	var loopObjs map[types.NamespacedName]bool
	switch e.TypeOfOwner.(type) {
	case *crd.ClowdApp:
		loopObjs = hashObj.ClowdApps
	case *crd.ClowdEnvironment:
		loopObjs = hashObj.ClowdEnvs
	}

	for k := range loopObjs {
		q.Add(reconcile.Request{NamespacedName: k})
	}
	return nil
}

func (e *enqueueRequestForObjectCustom) reconcileAllAppsUsingObject(ctx context.Context, obj client.Object, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	capps := &crd.ClowdAppList{}
	namespace := client.InNamespace(obj.GetNamespace())
	if err := e.client.List(ctx, capps, namespace); err != nil {
		e.logMessage(obj, err.Error(), "error listing apps", getNamespacedName(obj))
	}
	for _, app := range capps.Items {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Name: app.GetName(), Namespace: obj.GetNamespace()}})
	}
}

func (e *enqueueRequestForObjectCustom) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if evt.ObjectNew.GetAnnotations()[clowderconfig.LoadedConfig.Settings.RestarterAnnotationName] == "true" {
		shouldUpdate, err := e.updateHashCacheForConfigMapAndSecret(evt.ObjectNew)
		e.logMessage(evt.ObjectNew, "debug", fmt.Sprintf("shouldUpdate %s %v", e.ctrlName, shouldUpdate), getNamespacedName(evt.ObjectNew))
		if err != nil {
			e.logMessage(evt.ObjectNew, err.Error(), "", getNamespacedName(evt.ObjectNew))
		}
		if shouldUpdate {
			_ = e.doUpdateToHash(evt.ObjectNew, q)
		}
		if evt.ObjectOld.GetAnnotations()[clowderconfig.LoadedConfig.Settings.RestarterAnnotationName] != evt.ObjectNew.GetAnnotations()[clowderconfig.LoadedConfig.Settings.RestarterAnnotationName] {
			e.reconcileAllAppsUsingObject(ctx, evt.ObjectNew, q)
		}
	} else if _, err := e.hashCache.Read(evt.ObjectNew); err == nil {
		err := e.doUpdateToHash(evt.ObjectNew, q)
		if err != nil {
			e.hashCache.Delete(evt.ObjectNew)
		}
	}

	switch {
	case evt.ObjectNew != nil:
		if own, toKind := e.getOwner(evt.ObjectNew); own != nil {
			if doRequest, msg := e.HandlerFuncs.UpdateFunc(evt); doRequest {
				e.logMessage(evt.ObjectNew, msg, toKind, own)
				q.Add(reconcile.Request{NamespacedName: *own})
			}
		}
	case evt.ObjectOld != nil:
		if own, toKind := e.getOwner(evt.ObjectOld); own != nil {
			if doRequest, msg := e.HandlerFuncs.UpdateFunc(evt); doRequest {
				e.logMessage(evt.ObjectNew, msg, toKind, own)
				q.Add(reconcile.Request{NamespacedName: *own})
			}
		}
	}
}

func (e *enqueueRequestForObjectCustom) Delete(_ context.Context, evt event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.hashCache.Delete(evt.Object)

	if own, toKind := e.getOwner(evt.Object); own != nil {
		if doRequest, msg := e.HandlerFuncs.DeleteFunc(evt); doRequest {
			e.logMessage(evt.Object, msg, toKind, own)
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}

func (e *enqueueRequestForObjectCustom) Generic(_ context.Context, evt event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if own, toKind := e.getOwner(evt.Object); own != nil {
		if doRequest, msg := e.HandlerFuncs.GenericFunc(evt); doRequest {
			e.logMessage(evt.Object, msg, toKind, own)
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}
