package resolver

import (
	"errors"
	"testing"
)

func TestMerge_DistinctKindsAcrossExtensions(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
  kinds: [{name: widget}]
  interfaceTypes: [{name: http, targetKind: widget}]
`),
		refB: []byte(`
metadata: {id: ` + idB + `, version: v1alpha1}
spec:
  kinds: [{name: gadget}]
`),
	}
	g := mustResolve(t, docs, refA)
	c, err := Merge(g)
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}
	if !c.IsValidKind("widget") || !c.IsValidKind("gadget") {
		t.Fatalf("expected both kinds valid, got %v", c)
	}
	if !c.IsValidInterfaceType("widget", "http") {
		t.Fatal("expected widget/http interface type to be valid")
	}
	if c.IsValidKind("nope") {
		t.Fatal("expected unknown kind to be invalid")
	}
}

func TestMerge_AmbiguousKindIsAllowed(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
  kinds: [{name: widget}]
`),
		refB: []byte(`
metadata: {id: ` + idB + `, version: v1alpha1}
spec:
  kinds: [{name: widget}]
`),
	}
	g := mustResolve(t, docs, refA)
	c, err := Merge(g)
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}
	if !c.IsValidKind("widget") {
		t.Fatal("expected widget to be a valid kind")
	}
}

func TestMerge_ConflictingInterfaceTypeDeclarationFails(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
  kinds: [{name: widget}]
  interfaceTypes: [{name: http, targetKind: widget}]
`),
		refB: []byte(`
metadata: {id: ` + idB + `, version: v1alpha1}
spec:
  interfaceTypes: [{name: http, targetKind: widget}]
`),
	}
	g := mustResolve(t, docs, refA)
	_, err := Merge(g)
	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if conflictErr.InterfaceType != "http" {
		t.Fatalf("expected conflict on interface type http, got %q", conflictErr.InterfaceType)
	}
}

func TestMerge_ConflictingSchemaBindingFails(t *testing.T) {
	schemaDoc := `
  schemas:
    - id: widget-http
      appliesToKind: widget
      appliesToInterfaceType: http
      schema: {type: object}
`
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
  kinds: [{name: widget}]
  interfaceTypes: [{name: http, targetKind: widget}]
` + schemaDoc),
		refB: []byte(`
metadata: {id: ` + idB + `, version: v1alpha1}
spec:
` + schemaDoc),
	}
	g := mustResolve(t, docs, refA)
	_, err := Merge(g)
	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
}
