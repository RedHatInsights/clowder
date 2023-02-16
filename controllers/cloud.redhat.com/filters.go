package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/go-difflib/difflib"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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

func defaultFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	return HandlerFuncs{
		CreateFunc: func(e event.CreateEvent) (bool, string) {
			return true, "create"
		},
		DeleteFunc: func(e event.DeleteEvent) (bool, string) {
			return true, "update"
		},
		UpdateFunc: func(e event.UpdateEvent) (bool, string) {
			return true, "update"
		},
		GenericFunc: func(e event.GenericEvent) (bool, string) {
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

func configMapCreateFunc(e event.CreateEvent) (bool, string) {
	if checkForReconcile(e.Object) {
		doReconcile, _ := updateHashCache(e.Object)
		return doReconcile, "create"
	}
	return false, "create"
}

func configMapGenericFunc(e event.GenericEvent) (bool, string) {
	if checkForReconcile(e.Object) {
		doReconcile, _ := updateHashCache(e.Object)
		return doReconcile, "generic"
	}
	return false, "generic"
}

func configMapUpdateFunc(e event.UpdateEvent) (bool, string) {
	if e.ObjectNew.GetLabels()["watch"] == "me" {
		doReconcile, _ := updateHashCache(e.ObjectNew)
		return doReconcile, "update"
	}
	if e.ObjectNew.GetLabels()["watch"] != "me" {
		name := e.ObjectNew.GetName()
		namespace := e.ObjectNew.GetNamespace()
		DeleteHashCache(fmt.Sprintf("cm-%s-%s", name, namespace))
		return true, "update"
	}
	for _, owner := range e.ObjectNew.GetOwnerReferences() {
		if owner.Kind == "ClowdApp" && *owner.Controller {
			return true, "update"
		}
	}
	return false, "update"
}

func configMapDeleteFunc(e event.DeleteEvent) (bool, string) {
	name := e.Object.GetName()
	namespace := e.Object.GetNamespace()
	DeleteHashCache(fmt.Sprintf("cm-%s-%s", name, namespace))
	return true, "delete"
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

func configMapFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	filters := defaultFilter(logr, ctrlName)
	filters.GenericFunc = func(e event.GenericEvent) (bool, string) {
		return configMapGenericFunc(e)
	}
	filters.CreateFunc = func(e event.CreateEvent) (bool, string) {
		return configMapCreateFunc(e)
	}
	filters.UpdateFunc = func(e event.UpdateEvent) (bool, string) {
		return configMapUpdateFunc(e)
	}
	filters.DeleteFunc = func(e event.DeleteEvent) (bool, string) {
		return configMapDeleteFunc(e)
	}
	return filters
}

func alwaysFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	return genFilterFunc(func(e event.UpdateEvent) bool {
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

func environmentPredicate(logr logr.Logger, ctrlName string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: environmentUpdateFunc,
		GenericFunc: func(e event.GenericEvent) bool {
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
