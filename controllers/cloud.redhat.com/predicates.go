package controllers

import (
	"encoding/json"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	"github.com/RedHatInsights/go-difflib/difflib"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func isOursCreateFunc(logr logr.Logger, ctrlName string) func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
		if isOurs(e.Object, gvk) {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "create", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		}
		return false
	}
}

func isOursUpdateFunc(logr logr.Logger, ctrlName string) func(event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		if isOurs(e.ObjectNew, gvk) {
			displayUpdateDiff(e, logr, ctrlName, gvk)
			if res := e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration(); res {
				logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
				return true
			}
		}
		return false
	}
}

func isOursDeleteFunc(logr logr.Logger, ctrlName string) func(event.DeleteEvent) bool {
	return func(e event.DeleteEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
		if isOurs(e.Object, gvk) {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "delete", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		}
		return false
	}
}

func isOursGenericFunc(logr logr.Logger, ctrlName string) func(event.GenericEvent) bool {
	return func(e event.GenericEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
		if isOurs(e.Object, gvk) {
			logr.Info("Reconciliation trigger", "generic", ctrlName, "type", "delete", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		}
		return false
	}
}

func environmentPredicates(logr logr.Logger, ctrlName string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: isOursCreateFunc(logr, ctrlName),
		DeleteFunc: isOursDeleteFunc(logr, ctrlName),
		UpdateFunc: func(e event.UpdateEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
			if isOurs(e.ObjectNew, gvk) {
				if objOld, ok := e.ObjectOld.(*crd.ClowdEnvironment); ok {
					if objNew, ok := e.ObjectNew.(*crd.ClowdEnvironment); ok {
						displayUpdateDiff(e, logr, ctrlName, gvk)
						if !objOld.Status.Ready && objNew.Status.Ready {
							logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectOld.GetName(), "namespace", e.ObjectOld.GetNamespace())
							return true
						}
					}
				}
			}
			return false
		},
		GenericFunc: isOursGenericFunc(logr, ctrlName),
	}
}

func alwaysPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: isOursCreateFunc(logr, ctrlName),
		DeleteFunc: isOursDeleteFunc(logr, ctrlName),
		UpdateFunc: func(e event.UpdateEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
			if isOurs(e.ObjectNew, gvk) {
				displayUpdateDiff(e, logr, ctrlName, gvk)
				return true
			}
			return false
		},
		GenericFunc: isOursGenericFunc(logr, ctrlName),
	}
}

func kafkaPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: isOursCreateFunc(logr, ctrlName),
		DeleteFunc: isOursDeleteFunc(logr, ctrlName),
		UpdateFunc: func(e event.UpdateEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
			if isOurs(e.ObjectNew, gvk) {
				if objOld, ok := e.ObjectOld.(*strimzi.Kafka); ok {
					if objNew, ok := e.ObjectNew.(*strimzi.Kafka); ok {
						displayUpdateDiff(e, logr, ctrlName, gvk)
						if (objOld.Status != nil && objNew.Status != nil) && len(objOld.Status.Listeners) != len(objNew.Status.Listeners) {
							return true
						}
					}
				}
			}
			return false
		},
		GenericFunc: isOursGenericFunc(logr, ctrlName),
	}
}

func genericPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc:  isOursCreateFunc(logr, ctrlName),
		DeleteFunc:  isOursDeleteFunc(logr, ctrlName),
		UpdateFunc:  isOursUpdateFunc(logr, ctrlName),
		GenericFunc: isOursGenericFunc(logr, ctrlName),
	}
}

func displayUpdateDiff(e event.UpdateEvent, logr logr.Logger, ctrlName string, gvk schema.GroupVersionKind) {
	if clowderconfig.LoadedConfig.DebugOptions.Trigger.Diff {
		if e.ObjectNew.GetObjectKind().GroupVersionKind() == secretCompare {
			logr.Info("Trigger diff", "diff", "hidden", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectOld.GetName(), "namespace", e.ObjectOld.GetNamespace())
		} else {
			oldObjJSON, _ := json.MarshalIndent(e.ObjectOld, "", "  ")
			newObjJSON, _ := json.MarshalIndent(e.ObjectNew, "", "  ")

			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(oldObjJSON)),
				B:        difflib.SplitLines(string(newObjJSON)),
				FromFile: "old",
				ToFile:   "new",
				Context:  3,
			}
			text, _ := difflib.GetUnifiedDiffString(diff)
			logr.Info("Trigger diff", "diff", text, "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectOld.GetName(), "namespace", e.ObjectOld.GetNamespace())
		}
	}
}
