package controllers

import (
	"context"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

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
	context      context.Context
	hashCache    *hashcache.HashCache
}

var _ handler.EventHandler = &enqueueRequestForObjectCustom{}

var _ inject.Scheme = &enqueueRequestForObjectCustom{}

func (e *enqueueRequestForObjectCustom) InjectScheme(s *runtime.Scheme) error {
	return e.parseOwnerScheme(s)
}

var _ inject.Mapper = &enqueueRequestForObjectCustom{}

func (e *enqueueRequestForObjectCustom) InjectMapper(m meta.RESTMapper) error {
	e.mapObj = m
	return nil
}

var _ inject.Client = &enqueueRequestForObjectCustom{}

func (e *enqueueRequestForObjectCustom) InjectClient(c client.Client) error {
	e.client = c
	e.context = context.Background()
	return nil
}

func (e *enqueueRequestForObjectCustom) parseOwnerScheme(s *runtime.Scheme) error {
	kinds, _, err := s.ObjectKinds(e.TypeOfOwner)
	if err != nil {
		return err
	}
	e.groupKind = schema.GroupKind{Group: kinds[0].Group, Kind: kinds[0].Kind}
	return nil
}

func createNewHandler(input func(logr logr.Logger, ctrlName string) HandlerFuncs, log logr.Logger, ctrlName string, typeOfOwner runtime.Object, hashCache *hashcache.HashCache) handler.EventHandler {
	handleFuncs := input(log, ctrlName)
	obj := enqueueRequestForObjectCustom{
		HandlerFuncs: handleFuncs,
		TypeOfOwner:  typeOfOwner,
		logr:         log,
		ctrlName:     ctrlName,
		hashCache:    hashCache,
	}
	return &obj
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
		if obj.GetAnnotations()["qontract.reconcile"] == "true" {
			return e.hashCache.CreateOrUpdateObject(obj)
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

func (e *enqueueRequestForObjectCustom) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	shouldUpdate, err := e.updateHashCacheForConfigMapAndSecret(evt.Object)
	if err != nil {
		e.logMessage(evt.Object, err.Error(), "", getNamespacedName(evt.Object))
	}

	if shouldUpdate {
		_ = e.doUpdateToHash(evt.Object, q)
		e.reconcileAllAppsUsingObject(evt.Object, q)
	}

	if own, toKind := e.getOwner(evt.Object); own != nil {
		if doRequest, msg := e.HandlerFuncs.CreateFunc(evt); doRequest {
			e.logMessage(evt.Object, msg, toKind, own)
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}

func (e *enqueueRequestForObjectCustom) doUpdateToHash(obj client.Object, q workqueue.RateLimitingInterface) error {
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

func (e *enqueueRequestForObjectCustom) reconcileAllAppsUsingObject(obj client.Object, q workqueue.RateLimitingInterface) {
	capps := &crd.ClowdAppList{}
	if err := e.client.List(e.context, capps, client.InNamespace(obj.GetNamespace())); err != nil {
		e.logMessage(obj, err.Error(), "error listing apps", getNamespacedName(obj))
	}
	for _, app := range capps.Items {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Name: app.GetName(), Namespace: obj.GetNamespace()}})
	}
}

func (e *enqueueRequestForObjectCustom) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectNew.GetAnnotations()["qontract.reconcile"] == "true" {
		shouldUpdate, err := e.updateHashCacheForConfigMapAndSecret(evt.ObjectNew)
		e.logMessage(evt.ObjectNew, "debug", fmt.Sprintf("shouldUpdate %s %v", e.ctrlName, shouldUpdate), getNamespacedName(evt.ObjectNew))
		if err != nil {
			e.logMessage(evt.ObjectNew, err.Error(), "", getNamespacedName(evt.ObjectNew))
		}
		if shouldUpdate {
			_ = e.doUpdateToHash(evt.ObjectNew, q)
		}
		if evt.ObjectOld.GetAnnotations()["qontract.reconcile"] != evt.ObjectNew.GetAnnotations()["qontract.reconcile"] {
			e.reconcileAllAppsUsingObject(evt.ObjectNew, q)
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

func (e *enqueueRequestForObjectCustom) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.hashCache.Delete(evt.Object)

	if own, toKind := e.getOwner(evt.Object); own != nil {
		if doRequest, msg := e.HandlerFuncs.DeleteFunc(evt); doRequest {
			e.logMessage(evt.Object, msg, toKind, own)
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}

func (e *enqueueRequestForObjectCustom) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if own, toKind := e.getOwner(evt.Object); own != nil {
		if doRequest, msg := e.HandlerFuncs.GenericFunc(evt); doRequest {
			e.logMessage(evt.Object, msg, toKind, own)
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}
