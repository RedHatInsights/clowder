package controllers

import (
	"encoding/json"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/go-difflib/difflib"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

func logMessage(logr logr.Logger, ctrlName string, msg string, keysAndValues ...interface{}) {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		logr.Info(msg, keysAndValues...)
	}
}

func defaultPredicate(logr logr.Logger, ctrlName string) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
			logMessage(logr, ctrlName, "Reconciliation trigger", "ctrl", ctrlName, "type", "create", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
			logMessage(logr, ctrlName, "Reconciliation trigger", "ctrl", ctrlName, "type", "delete", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
			logMessage(logr, ctrlName, "Reconciliation trigger", "ctrl", ctrlName, "type", "create", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
			logMessage(logr, ctrlName, "Reconciliation trigger", "ctrl", ctrlName, "type", "generic", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		},
	}
}

func deploymentUpdateFunc(e event.UpdateEvent) bool {
	objOld := e.ObjectOld.(*apps.Deployment)
	objNew := e.ObjectNew.(*apps.Deployment)
	if objNew.GetGeneration() != objOld.GetGeneration() {
		return true
	}
	if (objOld.Status.AvailableReplicas != objNew.Status.AvailableReplicas) && (objNew.Status.AvailableReplicas == objNew.Status.ReadyReplicas) {
		return true
	}
	if (objOld.Status.AvailableReplicas == objOld.Status.ReadyReplicas) && (objNew.Status.AvailableReplicas != objNew.Status.ReadyReplicas) {
		return true
	}
	return false
}

func kafkaUpdateFunc(e event.UpdateEvent) bool {
	objOld := e.ObjectOld.(*strimzi.Kafka)
	objNew := e.ObjectNew.(*strimzi.Kafka)
	if (objOld.Status != nil && objNew.Status != nil) && len(objOld.Status.Listeners) != len(objNew.Status.Listeners) {
		return true
	}
	return false
}

func environmentUpdateFunc(e event.UpdateEvent) bool {
	objOld := e.ObjectOld.(*crd.ClowdEnvironment)
	objNew := e.ObjectNew.(*crd.ClowdEnvironment)
	if !objOld.Status.Ready && objNew.Status.Ready {
		return true
	}
	if objOld.GetGeneration() != objNew.GetGeneration() {
		return true
	}
	return false
}

func genPredicateFunc(updateFn func(e event.UpdateEvent) bool, logr logr.Logger, ctrlName string) predicate.Funcs {
	predicates := defaultPredicate(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		result := updateFn(e)
		if result {
			gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
			logMessage(logr, ctrlName, "Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			displayUpdateDiff(e, logr, ctrlName, gvk)
		}
		return result
	}
	return predicates
}

func alwaysPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return genPredicateFunc(func(e event.UpdateEvent) bool {
		return true
	}, logr, ctrlName)
}

func generationOnlyPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return genPredicateFunc(predicate.GenerationChangedPredicate{}.Update, logr, ctrlName)
}

func deploymentPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return genPredicateFunc(deploymentUpdateFunc, logr, ctrlName)
}

func kafkaPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return genPredicateFunc(kafkaUpdateFunc, logr, ctrlName)
}

func environmentPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return genPredicateFunc(environmentUpdateFunc, logr, ctrlName)
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
