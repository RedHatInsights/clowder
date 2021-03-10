package errors

import (
	"context"
	errlib "errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// ClowdKey is a string determining the type of error.
type ClowdKey string

// TODO: Make this configruable
var stacksEnabled bool = true

// ClowderError is a Clowder specific error, it has a number of functions attached to it to allow
// for creation and checking.
type ClowderError struct {
	Stack   zap.Field
	Msg     string
	Cause   error
	Requeue bool
}

func (a *ClowderError) Unwrap() error {
	return a.Cause
}

func (a *ClowderError) Error() string {
	for unwrapped := errlib.Unwrap(a); unwrapped != nil; unwrapped = errlib.Unwrap(unwrapped) {
		a.Msg = fmt.Sprintf("%s: %s", a.Msg, unwrapped.Error())
	}

	return a.Msg
}

// Is checks that a target is the same as a given error, that is, it has the same message and cause.
func (a *ClowderError) Is(target error) bool {
	b, ok := target.(*ClowderError)
	if !ok {
		return false
	}
	return (a.Msg == b.Msg && a.Cause == b.Cause)
}

// New constructs a new ClowderError object.
func New(msg string) *ClowderError {
	stackField := zap.String("stack", "")

	if stacksEnabled {
		stackField = zap.Stack("stack")
	}

	return &ClowderError{
		Msg:   msg,
		Stack: stackField,
	}
}

// Wrap takes an existing error an wraps it, returning a ClowderError
func Wrap(msg string, err error) *ClowderError {
	clowderErr := New(msg)
	clowderErr.Cause = err
	var cerr *ClowderError
	if errlib.As(err, &cerr) {
		clowderErr.Requeue = cerr.Requeue
	}
	return clowderErr
}

// MissingDependencies is a struct containing a map of string lists, it is intended to hold
// information about missing dependencies. The key is a resource type, or a provider as a string,
// and the list is a list of keys or items that are missing.
type MissingDependencies struct {
	MissingDeps map[string][]string
}

func (e *MissingDependencies) Error() string {
	typeList := []string{}

	for t, vals := range e.MissingDeps {
		depList := strings.Join(vals, ",")
		typeList = append(typeList, fmt.Sprintf("- %s: %s", t, depList))
	}

	body := strings.Join(typeList, ", ")

	return fmt.Sprintf("Missing dependencies: [%s]", body)
}

// RootCause takes an error an unwraps it, if it is nil, it calls RootCause on the returned err,
// this will recursively find an error that has an unwrapped value.
func RootCause(err error) error {
	cause := errlib.Unwrap(err)

	if cause != nil {
		return RootCause(cause)
	}

	return err
}

// GetRootStack will recurse through an error until it finds one with a stack string set.
func GetRootStack(err error) string {
	var stack string
	var clowderErr *ClowderError

	if errlib.As(err, &clowderErr) {
		cause := errlib.Unwrap(err)

		if cause != nil {
			stack = GetRootStack(cause)
		}

		if stack == "" {
			stack = clowderErr.Stack.String
		}
	}

	return stack
}

// LogError logs an error using the given contexts logger and a string.
func LogError(ctx context.Context, name string, err *ClowderError) {
	log := *(ctx.Value(ClowdKey("log")).(*logr.Logger))
	log.Error(err, err.Msg, "stack", GetRootStack(err))
}

// HandleError handles certain ClowdError types differently than normal errors.
func HandleError(ctx context.Context, err error) bool {
	log := *(ctx.Value(ClowdKey("log")).(*logr.Logger))
	recorder := *(ctx.Value(ClowdKey("recorder")).(*record.EventRecorder))
	obj := ctx.Value(ClowdKey("obj")).(runtime.Object)

	if err != nil {
		var depErr *MissingDependencies
		var clowderError *ClowderError
		if errlib.As(err, &depErr) {
			msg := depErr.Error()
			recorder.Event(obj, "Warning", "MissingDependencies", msg)
			log.Info(msg)
			return true
		} else if errlib.As(err, &clowderError) {
			msg := clowderError.Error()
			recorder.Event(obj, "Warning", "ClowdError", msg)
			log.Info(msg)
			if clowderError.Requeue {
				return true
			}
		}

		root := RootCause(err)
		if k8serr.IsConflict(root) {
			log.Info("Conflict reported.  Requeuing request.")
			return true
		}

		log.Error(err, "Reconciliation failure", "stack", GetRootStack(err))
	}
	return false
}
