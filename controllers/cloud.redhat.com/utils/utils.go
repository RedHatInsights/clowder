package utils

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const rCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

var Log logr.Logger = ctrllog.NullLogger{}

type ApplyError struct {
	Stack zap.Field
	Msg   string
}

func (a *ApplyError) Error() string {
	return fmt.Sprintf("%s:\n%s", a.Msg, a.Stack.String)
}

func NewApplyError(msg string) *ApplyError {
	return &ApplyError{
		Msg:   msg,
		Stack: zap.Stack("stack"),
	}
}

// RandString generates a random string of length n
func RandString(n int) string {
	b := make([]byte, n)

	for i := range b {
		b[i] = rCharSet[rand.Intn(len(rCharSet))]
	}

	return string(b)
}

type Updater bool

func (u *Updater) Apply(ctx context.Context, cl client.Client, obj runtime.Object) error {
	var err error
	var kind string

	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		kind = reflect.TypeOf(obj).String()
	} else {
		kind = obj.GetObjectKind().GroupVersionKind().Kind
	}

	if *u {
		Log.Info("Updating resource", "kind", kind)
		err = cl.Update(ctx, obj)
	} else {
		Log.Info("Creating resource", "kind", kind)
		err = cl.Create(ctx, obj)
	}

	return err
}

func RootCause(err error) error {
	cause := errors.Unwrap(err)

	if cause != nil {
		return RootCause(cause)
	}

	return err
}

func UpdateOrErr(err error) (Updater, error) {
	update := Updater(err == nil)

	if err != nil && !k8serr.IsNotFound(err) {
		return update, err
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
			return "", err
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

// MakeLabeler creates a function that will label objects with metadata from
// the given namespaced name and labels
func MakeLabeler(nn types.NamespacedName, labels map[string]string, env *crd.ClowdEnvironment) func(metav1.Object) {
	return func(o metav1.Object) {
		o.SetName(nn.Name)
		o.SetNamespace(nn.Namespace)
		o.SetLabels(labels)
		o.SetOwnerReferences([]metav1.OwnerReference{env.MakeOwnerReference()})
	}
}
