package utils

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const rCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

var Log logr.Logger = ctrllog.NullLogger{}

// RandString generates a random string of length n
func RandString(n int) string {
	b := make([]byte, n)

	for i := range b {
		b[i] = rCharSet[rand.Intn(len(rCharSet))]
	}

	return string(b)
}

type Updater bool

type PClient interface {
	client.Client
	AddResource(runtime.Object)
}

func (u *Updater) Apply(ctx context.Context, cl client.Client, obj runtime.Object) error {
	var err error
	var kind string

	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		kind = reflect.TypeOf(obj).String()
	} else {
		kind = obj.GetObjectKind().GroupVersionKind().Kind
	}

	meta := obj.(metav1.Object)

	if *u {
		Log.Info("Updating resource", "namespace", meta.GetNamespace(), "name", meta.GetName(), "kind", kind)
		err = cl.Update(ctx, obj)
	} else {
		if meta.GetName() == "" {
			Log.Info("Skipping resource as name unknown", "kind", kind)
			return nil
		}

		Log.Info("Creating resource", "namespace", meta.GetNamespace(), "name", meta.GetName(), "kind", kind)
		err = cl.Create(ctx, obj)
	}

	if err != nil {
		verb := "creating"
		if *u {
			verb = "updating"
		}

		msg := fmt.Sprintf("Error %s resource %s %s", verb, kind, meta.GetName())
		return errors.Wrap(msg, err)
	}

	return nil
}

func UpdateOrErr(err error) (Updater, error) {
	update := Updater(err == nil)

	if err != nil && !k8serr.IsNotFound(err) {
		return update, errors.Wrap("Failed to update", err)
	}

	return update, nil
}

func UpdateAllOrErr(ctx context.Context, cl client.Client, nn types.NamespacedName, obj ...runtime.Object) (map[runtime.Object]Updater, error) {
	updates := map[runtime.Object]Updater{}

	for _, resource := range obj {
		update, err := UpdateOrErr(cl.Get(ctx, nn, resource))

		if err != nil {
			return updates, err
		}

		updates[resource] = update
	}

	return updates, nil
}

func ApplyAll(ctx context.Context, cl client.Client, updates map[runtime.Object]Updater) error {
	for resource, update := range updates {
		if err := update.Apply(ctx, cl, resource); err != nil {
			return err
		}
	}

	return nil
}

// B64Decode decodes the provided secret
func B64Decode(s *core.Secret, key string) (string, error) {
	decoded, err := b64.StdEncoding.DecodeString(string(s.Data[key]))

	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func IntMinMax(listStrInts []string, max bool) (string, error) {
	var listInts []int
	for _, strint := range listStrInts {
		i, err := strconv.Atoi(strint)
		if err != nil {
			return "", errors.Wrap("Failed to convert", err)
		}
		listInts = append(listInts, i)
	}
	ol := listInts[0]
	for i, e := range listInts {
		if max {
			if i == 0 || e > ol {
				ol = e
			}
		} else {
			if i == 0 || e < ol {
				ol = e
			}
		}
	}
	return strconv.Itoa(ol), nil
}

func IntMin(listStrInts []string) (string, error) {
	return IntMinMax(listStrInts, false)
}

func IntMax(listStrInts []string) (string, error) {
	return IntMinMax(listStrInts, true)
}

func ListMerge(listStrs []string) (string, error) {
	optionStrings := make(map[string]bool)
	for _, optionsList := range listStrs {
		brokenString := strings.Split(optionsList, ",")
		for _, option := range brokenString {
			optionStrings[strings.TrimSpace(option)] = true
		}
	}
	keys := make([]string, len(optionStrings))

	i := 0
	for key := range optionStrings {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return strings.Join(keys, ","), nil
}

// Int32 returns a pointer to an int32 version of n
func Int32(n int) *int32 {
	t := int32(n)
	return &t
}

// PointTrue returns a pointer to True
func PointTrue() *bool {
	t := true
	return &t
}

// MakeLabeler creates a function that will label objects with metadata from
// the given namespaced name and labels
func MakeLabeler(nn types.NamespacedName, labels map[string]string, obj obj.ClowdObject) func(metav1.Object) {
	return func(o metav1.Object) {
		o.SetName(nn.Name)
		o.SetNamespace(nn.Namespace)
		o.SetLabels(labels)
		o.SetOwnerReferences([]metav1.OwnerReference{obj.MakeOwnerReference()})
	}
}

func GetCustomLabeler(labels map[string]string, nn types.NamespacedName, baseResource obj.ClowdObject) func(metav1.Object) {
	appliedLabels := baseResource.GetLabels()
	if labels != nil {
		for k, v := range labels {
			appliedLabels[k] = v
		}
	}
	return MakeLabeler(nn, appliedLabels, baseResource)
}

func MakeService(service *core.Service, nn types.NamespacedName, labels map[string]string, ports []core.ServicePort, baseResource obj.ClowdObject) {
	labeler := GetCustomLabeler(labels, nn, baseResource)
	labeler(service)
	service.Spec.Selector = labels
	service.Spec.Ports = ports
}

func MakePVC(pvc *core.PersistentVolumeClaim, nn types.NamespacedName, labels map[string]string, size string, baseResource obj.ClowdObject) {
	labeler := GetCustomLabeler(labels, nn, baseResource)
	labeler(pvc)
	pvc.Spec.AccessModes = []core.PersistentVolumeAccessMode{core.ReadWriteOnce}
	pvc.Spec.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceName(core.ResourceStorage): resource.MustParse(size),
		},
	}
}

func DeploymentStatusChecker(deployment *apps.Deployment) bool {
	if deployment.Generation <= deployment.Status.ObservedGeneration {
		if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			return false
		}
		if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
			return false
		}
		if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
			return false
		}
		return true
	}
	return false
}
