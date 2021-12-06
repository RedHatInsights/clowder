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

func defaultPredicateLog(logr logr.Logger, ctrlName string) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "create", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "delete", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "create", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			gvk, _ := utils.GetKindFromObj(Scheme, e.Object)
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "generic", "resType", gvk.Kind, "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())
			return true
		},
	}
}

func generationUpdateFunc(e event.UpdateEvent) bool {
	if res := e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration(); res {
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
	return false
}

// These functions only return if the generation changes
func getGenerationOnlyPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return generationOnlyPredicateWithLog(logr, ctrlName)
	}
	return generationOnlyPredicate
}

var generationOnlyPredicate = predicate.Funcs{
	UpdateFunc: generationUpdateFunc,
}

func generationOnlyPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		displayUpdateDiff(e, logr, ctrlName, gvk)
		result := generationUpdateFunc(e)
		if result {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
		}
		return result
	}
	return predicates
}

// These functions always return on an update
func getAlwaysPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return alwaysPredicateWithLog(logr, ctrlName)
	}
	return predicate.Funcs{}
}

func alwaysPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		displayUpdateDiff(e, logr, ctrlName, gvk)
		logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
		return true
	}
	return predicates
}

//These functions are specific to Kafka
func getKafkaPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return kafkaPredicateWithLog(logr, ctrlName)
	}
	return kafkaPredicate
}

var kafkaPredicate = predicate.Funcs{
	UpdateFunc: kafkaUpdateFunc,
}

func kafkaPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		displayUpdateDiff(e, logr, ctrlName, gvk)
		result := kafkaUpdateFunc(e)
		if result {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
		}
		return result
	}
	return predicates
}

//These functions are specific to ClowdEnvironment
func getEnvironmentPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return environmentPredicateWithLog(logr, ctrlName)
	}
	return environmentPredicate
}

var environmentPredicate = predicate.Funcs{
	UpdateFunc: environmentUpdateFunc,
}

func environmentPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		displayUpdateDiff(e, logr, ctrlName, gvk)
		result := environmentUpdateFunc(e)
		if result {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
		}
		return result
	}
	return predicates
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
