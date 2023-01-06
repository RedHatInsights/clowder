package controllers

import (
	"fmt"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	"github.com/go-logr/logr"
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
)

type enqueueRequestForObjectCustom struct {
	HandlerFuncs HandlerFuncs
	groupKind    schema.GroupKind
	TypeOfOwner  runtime.Object
	mapObj       meta.RESTMapper
	logr         logr.Logger
	ctrlName     string
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

func (e *enqueueRequestForObjectCustom) parseOwnerScheme(s *runtime.Scheme) error {
	kinds, _, err := s.ObjectKinds(e.TypeOfOwner)
	if err != nil {
		return err
	}
	e.groupKind = schema.GroupKind{Group: kinds[0].Group, Kind: kinds[0].Kind}
	return nil
}

func createNewHandler(input func(logr logr.Logger, ctrlName string) HandlerFuncs, log logr.Logger, ctrlName string, typeOfOwner runtime.Object) handler.EventHandler {
	handleFuncs := input(log, ctrlName)
	obj := enqueueRequestForObjectCustom{
		HandlerFuncs: handleFuncs,
		TypeOfOwner:  typeOfOwner,
		logr:         log,
		ctrlName:     ctrlName,
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

func (e *enqueueRequestForObjectCustom) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	own, toKind := e.findOwner(evt.Object)
	if own == nil {
		return
	}
	doRequest, msg := e.HandlerFuncs.CreateFunc(evt)
	if doRequest {
		gvk, _ := utils.GetKindFromObj(Scheme, evt.Object)
		logMessage(e.logr, "Reconciliation trigger", "ctrl", e.ctrlName, "type", msg, "resType", gvk.Kind, "fromObj", fmt.Sprintf("%s/%s/%s", gvk.Kind, evt.Object.GetNamespace(), evt.Object.GetName()), "toObj", fmt.Sprintf("%s/%s/%s", toKind, own.Namespace, own.Name))
		q.Add(reconcile.Request{NamespacedName: *own})
	}
}

func (e *enqueueRequestForObjectCustom) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	switch {
	case evt.ObjectNew != nil:
		own, toKind := e.findOwner(evt.ObjectNew)
		if own == nil {
			return
		}
		doRequest, msg := e.HandlerFuncs.UpdateFunc(evt)
		if doRequest {
			gvk, _ := utils.GetKindFromObj(Scheme, evt.ObjectNew)
			logMessage(e.logr, "Reconciliation trigger", "ctrl", e.ctrlName, "type", msg, "resType", gvk.Kind, "fromObj", fmt.Sprintf("%s/%s/%s", gvk.Kind, evt.ObjectNew.GetNamespace(), evt.ObjectNew.GetName()), "toObj", fmt.Sprintf("%s/%s/%s", toKind, own.Namespace, own.Name))
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	case evt.ObjectOld != nil:
		own, toKind := e.findOwner(evt.ObjectOld)
		if own == nil {
			return
		}
		doRequest, msg := e.HandlerFuncs.UpdateFunc(evt)
		if doRequest {
			gvk, _ := utils.GetKindFromObj(Scheme, evt.ObjectOld)
			logMessage(e.logr, "Reconciliation trigger", "ctrl", e.ctrlName, "type", msg, "resType", gvk.Kind, "fromObj", fmt.Sprintf("%s/%s/%s", gvk.Kind, evt.ObjectOld.GetNamespace(), evt.ObjectOld.GetName()), "toObj", fmt.Sprintf("%s/%s/%s", toKind, own.Namespace, own.Name))
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}

func (e *enqueueRequestForObjectCustom) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	own, toKind := e.findOwner(evt.Object)
	if own == nil {
		return
	}
	doRequest, msg := e.HandlerFuncs.DeleteFunc(evt)
	if doRequest {
		gvk, _ := utils.GetKindFromObj(Scheme, evt.Object)
		logMessage(e.logr, "Reconciliation trigger", "ctrl", e.ctrlName, "type", msg, "resType", gvk.Kind, "fromObj", fmt.Sprintf("%s/%s/%s", gvk.Kind, evt.Object.GetNamespace(), evt.Object.GetName()), "toObj", fmt.Sprintf("%s/%s/%s", toKind, own.Namespace, own.Name))
		q.Add(reconcile.Request{NamespacedName: *own})
	}
}

func (e *enqueueRequestForObjectCustom) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}
	own, toKind := e.findOwner(evt.Object)
	if own == nil {
		return
	}
	doRequest, msg := e.HandlerFuncs.GenericFunc(evt)
	if doRequest {
		gvk, _ := utils.GetKindFromObj(Scheme, evt.Object)
		logMessage(e.logr, "Reconciliation trigger", "ctrl", e.ctrlName, "type", msg, "resType", gvk.Kind, "fromObj", fmt.Sprintf("%s/%s/%s", gvk.Kind, evt.Object.GetNamespace(), evt.Object.GetName()), "toObj", fmt.Sprintf("%s/%s/%s", toKind, own.Namespace, own.Name))
		q.Add(reconcile.Request{NamespacedName: *own})
	}
}
