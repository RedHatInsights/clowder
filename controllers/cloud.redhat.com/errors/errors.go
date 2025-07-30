// Package errors provides custom error types and error handling utilities for Clowder
package errors

import (
	"context"
	errlib "errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
)

// ClowdKey is a string determining the type of error.
type ClowdKey string

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
	return a.Msg == b.Msg && a.Cause == b.Cause
}

// NewClowderError constructs a new ClowderError object.
func NewClowderError(msg string) *ClowderError {
	stackField := zap.Stack("stack")

	return &ClowderError{
		Msg:   msg,
		Stack: stackField,
	}
}

// Wrap takes an existing error an wraps it, returning a ClowderError
func Wrap(msg string, err error) *ClowderError {
	clowderErr := NewClowderError(msg)
	clowderErr.Cause = err
	var cerr *ClowderError
	if errlib.As(err, &cerr) {
		clowderErr.Requeue = cerr.Requeue
	}
	return clowderErr
}

// MissingDependency is a struct that holds information about a missing dependency
type MissingDependency struct {
	Source  string
	Details string
}

// ToString returns a string representation of the missing dependency
func (m *MissingDependency) ToString() string {
	return fmt.Sprintf("source: %s, details: %s", m.Source, m.Details)
}

// MakeMissingDependencies creates a MissingDependencies error from a single MissingDependency
func MakeMissingDependencies(missingDep MissingDependency) MissingDependencies {
	return MissingDependencies{
		MissingDeps: []MissingDependency{missingDep},
	}
}

// MissingDependencies is a struct that holds a list of MissingDependency structs
type MissingDependencies struct {
	MissingDeps []MissingDependency
}

// Error returns a string representation of the missing dependencies
func (e *MissingDependencies) Error() string {
	typeList := []string{}

	for _, missingDep := range e.MissingDeps {
		typeList = append(typeList, missingDep.ToString())
	}

	body := strings.Join(typeList, "; ")

	return fmt.Sprintf("Missing dependencies: [%s]", body)
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
func LogError(ctx context.Context, err *ClowderError) {
	log := *(ctx.Value(ClowdKey("log")).(*logr.Logger))
	log.Error(err, err.Msg, "stack", GetRootStack(err))
}
