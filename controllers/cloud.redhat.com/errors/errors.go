package errors

import (
	errlib "errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
)

// TODO: Make this configruable
var stacksEnabled bool = true

type ClowderError struct {
	Stack zap.Field
	Msg   string
	Cause error
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

func (a *ClowderError) Is(target error) bool {
	b, ok := target.(*ClowderError)
	if !ok {
		return false
	}
	return (a.Msg == b.Msg && a.Cause == b.Cause)
}

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

func Wrap(msg string, err error) error {
	clowderErr := New(msg)
	clowderErr.Cause = err
	return clowderErr
}

type MissingDependencies struct {
	MissingDeps map[string][]string
}

func (e *MissingDependencies) Error() string {
	typeList := []string{}

	for t, vals := range e.MissingDeps {
		depList := strings.Join(vals, "\n\t")
		typeList = append(typeList, fmt.Sprintf("%s\n\t%s", t, depList))
	}

	body := strings.Join(typeList, "\n")

	return fmt.Sprintf("Missing dependencies: \n%s", body)
}

func RootCause(err error) error {
	cause := errlib.Unwrap(err)

	if cause != nil {
		return RootCause(cause)
	}

	return err
}

func HandleError(log logr.Logger, err error) bool {
	if err != nil {
		var depErr *MissingDependencies

		if errlib.As(err, &depErr) {
			// TODO: emit event
			log.Info(depErr.Error())
			return true
		}

		root := RootCause(err)
		if k8serr.IsConflict(root) {
			log.Info("Conflict reported.  Requeuing request.")
			return true
		}

		var clowderErr *ClowderError
		if errlib.As(err, &clowderErr) {
			log.Error(err, "Reconciliation failure", "stack", clowderErr.Stack.String)
		} else {
			log.Error(err, "Reconciliation failure")
		}
	}
	return false
}
