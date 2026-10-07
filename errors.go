package resolver

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNotFound = errors.New("extension not found")

type FetchError struct {
	Reference ExtensionReference
	Err       error
}

func (e *FetchError) Error() string {
	return fmt.Sprintf("fetch %s: %v", e.Reference, e.Err)
}

func (e *FetchError) Unwrap() error { return e.Err }

type CycleError struct {
	Path []ExtensionReference
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("dependency cycle: %s", strings.Join(referenceStrings(e.Path), " -> "))
}

// InterfaceType is empty when the conflict is on a kind itself.
type ConflictError struct {
	Kind          string
	InterfaceType string
	DeclaredBy    []ExtensionReference
}

func (e *ConflictError) Error() string {
	subject := e.Kind
	if e.InterfaceType != "" {
		subject = fmt.Sprintf("%s/%s", e.Kind, e.InterfaceType)
	}
	return fmt.Sprintf("conflict on %q: declared by %s", subject, strings.Join(referenceStrings(e.DeclaredBy), ", "))
}

// ExtensionMismatchError is returned when a Condition names an extension
// that isn't one of the resolved extensions defining its kind.
type ExtensionMismatchError struct {
	Kind      string
	Extension ExtensionReference
}

func (e *ExtensionMismatchError) Error() string {
	return fmt.Sprintf("extension %s does not define kind %q", e.Extension, e.Kind)
}

func referenceStrings(references []ExtensionReference) []string {
	values := make([]string, len(references))
	for i, reference := range references {
		values[i] = reference.String()
	}
	return values
}
