package errors

import (
	"context"
	errlib "errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClowdKey is a string determining the type of error.
type ClowdKey string

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
	var causeMsg = ""
	if cause := a.Unwrap(); cause != nil {
		causeMsg = cause.Error()
	}
	return a.Msg + ": " + causeMsg
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

//MissingDependency is a struct that holds information about a missing dependency
type MissingDependency struct {
	Source  string
	Details string
}

//ToString returns a string representation of the missing dependency
func (m *MissingDependency) ToString() string {
	return fmt.Sprintf("source: %s, details: %s", m.Source, m.Details)
}

func MakeMissingDependencies(missingDep MissingDependency) MissingDependencies {
	return MissingDependencies{
		MissingDeps: []MissingDependency{missingDep},
	}
}

//MissingDependencies is a struct that holds a list of MissingDependency structs
type MissingDependencies struct {
	MissingDeps []MissingDependency
}

//Error returns a string representation of the missing dependencies
func (e *MissingDependencies) Error() string {
	typeList := []string{}

	for _, missingDep := range e.MissingDeps {
		typeList = append(typeList, missingDep.ToString())
	}

	body := strings.Join(typeList, "; ")

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
	obj := ctx.Value(ClowdKey("obj")).(client.Object)

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
