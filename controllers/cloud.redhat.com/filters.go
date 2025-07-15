package controllers

import (
	"encoding/json"

	"github.com/RedHatInsights/go-difflib/difflib"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"

	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

type HandlerFuncs struct {
	// Create returns true if the Create event should be processed
	CreateFunc func(event.CreateEvent) (bool, string)

	// Delete returns true if the Delete event should be processed
	DeleteFunc func(event.DeleteEvent) (bool, string)

	// Update returns true if the Update event should be processed
	UpdateFunc func(event.UpdateEvent) (bool, string)

	// Generic returns true if the Generic event should be processed
	GenericFunc func(event.GenericEvent) (bool, string)
}

func logMessage(logr logr.Logger, msg string, keysAndValues ...interface{}) {
	logr.Info(msg, keysAndValues...)
}

func defaultFilter(_ logr.Logger, _ string) HandlerFuncs {
	return HandlerFuncs{
		CreateFunc: func(_ event.CreateEvent) (bool, string) {
			return true, "create"
		},
		DeleteFunc: func(_ event.DeleteEvent) (bool, string) {
			return true, "update"
		},
		UpdateFunc: func(_ event.UpdateEvent) (bool, string) {
			return true, "update"
		},
		GenericFunc: func(_ event.GenericEvent) (bool, string) {
			return true, "generic"
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

func genFilterFunc(updateFn func(e event.UpdateEvent) bool, logr logr.Logger, ctrlName string) HandlerFuncs {
	filters := defaultFilter(logr, ctrlName)
	filters.UpdateFunc = func(e event.UpdateEvent) (bool, string) {
		result := updateFn(e)
		if result {
			gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
			displayUpdateDiff(e, logr, ctrlName, gvk)
		}
		return result, "update"
	}
	return filters
}

func alwaysFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	return genFilterFunc(func(_ event.UpdateEvent) bool {
		return true
	}, logr, ctrlName)
}

func generationOnlyFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	return genFilterFunc(predicate.GenerationChangedPredicate{}.Update, logr, ctrlName)
}

func deploymentFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	return genFilterFunc(deploymentUpdateFunc, logr, ctrlName)
}

func kafkaFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	return genFilterFunc(kafkaUpdateFunc, logr, ctrlName)
}

func environmentPredicate(_ logr.Logger, _ string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(_ event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: environmentUpdateFunc,
		GenericFunc: func(_ event.GenericEvent) bool {
			return true
		},
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
