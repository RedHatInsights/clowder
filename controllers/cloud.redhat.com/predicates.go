package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	"github.com/RedHatInsights/go-difflib/difflib"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// These functions only return if the generation changes
func getGenerationOnlyPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return generationOnlyPredicateWithLog(logr, ctrlName)
	}
	return predicate.GenerationChangedPredicate{}
}

func generationOnlyPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	genPredicate := predicate.GenerationChangedPredicate{}
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		result := genPredicate.Update(e)
		if result {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			displayUpdateDiff(e, logr, ctrlName, gvk)
		}
		return result
	}
	return predicates
}

// These functions are returned for deployments
// These functions always return on an update
func getDeploymentPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return deploymentPredicateWithLog(logr, ctrlName)
	}
	return predicate.Funcs{
		UpdateFunc: deploymentUpdateFunc,
	}
}

func deploymentPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		result := deploymentUpdateFunc(e)
		if result {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			displayUpdateDiff(e, logr, ctrlName, gvk)
			return true
		}
		return false
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
		logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
		displayUpdateDiff(e, logr, ctrlName, gvk)
		return true
	}
	return predicates
}

//These functions are specific to Kafka
func getKafkaPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return kafkaPredicateWithLog(logr, ctrlName)
	}
	return predicate.Funcs{
		UpdateFunc: kafkaUpdateFunc,
	}
}

func kafkaPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		result := kafkaUpdateFunc(e)
		if result {
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
			displayUpdateDiff(e, logr, ctrlName, gvk)
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
	return predicate.Funcs{
		UpdateFunc: environmentUpdateFunc,
	}
}

func environmentPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		gvk, _ := utils.GetKindFromObj(Scheme, e.ObjectNew)
		result := environmentUpdateFunc(e)
		if result {
			displayUpdateDiff(e, logr, ctrlName, gvk)
			logr.Info("Reconciliation trigger", "ctrl", ctrlName, "type", "update", "resType", gvk.Kind, "name", e.ObjectNew.GetName(), "namespace", e.ObjectNew.GetNamespace())
		}
		return result
	}
	return predicates
}

func updateHashCache(obj client.Object) (bool, error) {
	gvk, err := utils.GetKindFromObj(Scheme, obj)
	if err != nil {
		return true, err
	}

	var jsonData []byte
	var prefix string
	switch gvk.Kind {
	case "ConfigMap":
		cm := &core.ConfigMap{}
		prefix = "cm"
		err = Scheme.Convert(obj, cm, context.Background())

		if err != nil {
			return true, fmt.Errorf("couldn't convert: %s", err)
		}
		jsonData, err = json.Marshal(cm.Data)
		if err != nil {
			return true, nil
		}
	case "Secret":
		s := &core.Secret{}
		prefix = "sc"
		err = Scheme.Convert(obj, s, context.Background())

		if err != nil {
			return true, fmt.Errorf("couldn't convert: %s", err)
		}
		jsonData, err = json.Marshal(s.Data)
		if err != nil {
			return true, nil
		}
	}

	h := sha256.New()
	h.Write([]byte(jsonData))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	currentHash := ReadHashCache(fmt.Sprintf("%s-%s-%s", prefix, obj.GetName(), obj.GetNamespace()))

	if currentHash == hash {
		return false, nil
	}

	SetHashCache(fmt.Sprintf("%s-%s-%s", prefix, obj.GetName(), obj.GetNamespace()), hash)

	return true, nil
}

func checkForReconcile(obj client.Object) bool {
	if obj.GetLabels()["watch"] == "me" {
		return true
	}
	for _, owner := range obj.GetOwnerReferences() {
		if owner.Kind == "ClowdApp" && *owner.Controller {
			return true
		}
	}
	return false
}

func configMapCreateFunc(e event.CreateEvent) bool {
	if checkForReconcile(e.Object) {
		doReconcile, _ := updateHashCache(e.Object)
		return doReconcile
	}
	return false
}

func configMapGenericFunc(e event.GenericEvent) bool {
	if checkForReconcile(e.Object) {
		doReconcile, _ := updateHashCache(e.Object)
		return doReconcile
	}
	return false
}

func configMapUpdateFunc(e event.UpdateEvent) bool {
	if e.ObjectNew.GetLabels()["watch"] == "me" {
		doReconcile, _ := updateHashCache(e.ObjectNew)
		return doReconcile
	}
	if e.ObjectNew.GetLabels()["watch"] != "me" {
		name := e.ObjectNew.GetName()
		namespace := e.ObjectNew.GetNamespace()
		DeleteHashCache(fmt.Sprintf("cm-%s-%s", name, namespace))
		return true
	}
	for _, owner := range e.ObjectNew.GetOwnerReferences() {
		if owner.Kind == "ClowdApp" && *owner.Controller {
			return true
		}
	}
	return false
}

func configMapDeleteFunc(e event.DeleteEvent) bool {
	name := e.Object.GetName()
	namespace := e.Object.GetNamespace()
	DeleteHashCache(fmt.Sprintf("cm-%s-%s", name, namespace))
	return true
}

// These functions are for configmaps return on an update
func getConfigMapOrSecretPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	if clowderconfig.LoadedConfig.DebugOptions.Logging.DebugLogging {
		return configmapPredicateWithLog(logr, ctrlName)
	}
	return predicate.Funcs{
		CreateFunc:  configMapCreateFunc,
		DeleteFunc:  configMapDeleteFunc,
		UpdateFunc:  configMapUpdateFunc,
		GenericFunc: configMapGenericFunc,
	}
}

func configmapPredicateWithLog(logr logr.Logger, ctrlName string) predicate.Predicate {
	predicates := defaultPredicateLog(logr, ctrlName)
	predicates.GenericFunc = func(e event.GenericEvent) bool {
		return configMapGenericFunc(e)
	}
	predicates.CreateFunc = func(e event.CreateEvent) bool {
		return configMapCreateFunc(e)
	}
	predicates.UpdateFunc = func(e event.UpdateEvent) bool {
		return configMapUpdateFunc(e)
	}
	predicates.DeleteFunc = func(e event.DeleteEvent) bool {
		return configMapDeleteFunc(e)
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
