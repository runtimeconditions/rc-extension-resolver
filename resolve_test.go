package resolver

import (
	"errors"
	"testing"
)

const (
	idA = "example.a"
	idB = "mem://b.extension.yaml"
	idC = "mem://c.extension.yaml"
)

var (
	refA = ExtensionReference{ID: idA, Version: "v1alpha1"}
	refB = ExtensionReference{ID: idB, Version: "v1alpha1"}
	refC = ExtensionReference{ID: idC, Version: "v1alpha1"}
)

func mustResolve(t *testing.T, docs map[ExtensionReference][]byte, root ExtensionReference) *ResolvedGraph {
	t.Helper()
	r := NewResolver(NewInMemoryLoader(docs))
	g, err := r.Resolve(root)
	if err != nil {
		t.Fatalf("Resolve(%s): unexpected error: %v", root, err)
	}
	return g
}

func TestResolve_SingleExtensionNoDependencies(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
apiVersion: runtimeconditions.io/v1alpha1
kind: RuntimeConditionsExtensionDefinition
metadata:
  id: ` + idA + `
  version: v1alpha1
spec:
  kinds:
    - name: widget
`),
	}
	g := mustResolve(t, docs, refA)
	if len(g.Extensions) != 1 {
		t.Fatalf("expected 1 extension, got %d", len(g.Extensions))
	}
	if g.ByID[refA] == nil {
		t.Fatalf("expected %s in ByID", idA)
	}
}

func TestResolve_TransitiveDependenciesInTopologicalOrder(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata:
  id: ` + idA + `
  version: v1alpha1
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
  kinds: [{name: a}]
`),
		refB: []byte(`
metadata:
  id: ` + idB + `
  version: v1alpha1
spec:
  dependencies: [{id: ` + idC + `, version: v1alpha1}]
  kinds: [{name: b}]
`),
		refC: []byte(`
metadata:
  id: ` + idC + `
  version: v1alpha1
spec:
  kinds: [{name: c}]
`),
	}
	g := mustResolve(t, docs, refA)
	if len(g.Extensions) != 3 {
		t.Fatalf("expected 3 extensions, got %d", len(g.Extensions))
	}
	// dependency-first: c before b before a
	order := map[ExtensionReference]int{}
	for i, ext := range g.Extensions {
		order[ext.Metadata.Reference()] = i
	}
	if !(order[refC] < order[refB] && order[refB] < order[refA]) {
		t.Fatalf("expected topological order c,b,a; got order %v", order)
	}
}

func TestResolve_DiamondDependencyFetchedOnce(t *testing.T) {
	// Two releases of a share d through a diamond, without forming a cycle.
	refB := ExtensionReference{ID: idA, Version: "v2"}
	idD := "mem://d.extension.yaml"
	refD := ExtensionReference{ID: idD, Version: "v1alpha1"}
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idA + `, version: v2}, {id: ` + idC + `, version: v1alpha1}]
  kinds: [{name: a}]
`),
		refB: []byte(`
metadata: {id: ` + idA + `, version: v2}
spec:
  dependencies: [{id: ` + idD + `, version: v1alpha1}]
  kinds: [{name: b}]
`),
		refC: []byte(`
metadata: {id: ` + idC + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idD + `, version: v1alpha1}]
  kinds: [{name: c}]
`),
		refD: []byte(`
metadata: {id: ` + idD + `, version: v1alpha1}
spec:
  kinds: [{name: d}]
`),
	}
	g := mustResolve(t, docs, refA)
	if len(g.Extensions) != 4 {
		t.Fatalf("expected 4 unique extensions, got %d: %v", len(g.Extensions), g.Extensions)
	}
}

func TestResolve_BrokenReferenceFails(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
  kinds: [{name: a}]
`),
		// idB deliberately missing.
	}
	r := NewResolver(NewInMemoryLoader(docs))
	_, err := r.Resolve(refA)
	if err == nil {
		t.Fatal("expected error for broken reference, got nil")
	}
	var fetchErr *FetchError
	if !errors.As(err, &fetchErr) {
		t.Fatalf("expected *FetchError, got %T: %v", err, err)
	}
	if fetchErr.Reference != refB {
		t.Fatalf("expected fetch error for %s, got %s", idB, fetchErr.Reference)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected error to wrap ErrNotFound, got %v", err)
	}
}

func TestResolve_DirectCycleFails(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
`),
		refB: []byte(`
metadata: {id: ` + idB + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idA + `, version: v1alpha1}]
`),
	}
	r := NewResolver(NewInMemoryLoader(docs))
	_, err := r.Resolve(refA)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected *CycleError, got %T: %v", err, err)
	}
}

func TestResolve_SelfCycleFails(t *testing.T) {
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idA + `, version: v1alpha1}]
`),
	}
	r := NewResolver(NewInMemoryLoader(docs))
	_, err := r.Resolve(refA)
	var cycleErr *CycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected *CycleError, got %T: %v", err, err)
	}
}

func TestResolve_MismatchedMetadataFails(t *testing.T) {
	for _, metadata := range []string{
		"{id: " + idB + ", version: v1alpha1}",
		"{id: " + idA + ", version: v2}",
	} {
		t.Run(metadata, func(t *testing.T) {
			docs := map[ExtensionReference][]byte{
				refA: []byte("metadata: " + metadata + "\nspec: {}\n"),
			}
			r := NewResolver(NewInMemoryLoader(docs))
			if _, err := r.Resolve(refA); err == nil {
				t.Fatal("expected error for mismatched metadata.id or metadata.version")
			}
		})
	}
}

// An extension release that depends on another release, four levels deep,
// with only the next release declared at each level - the resolver
// shouldn't need to know the chain's depth up front.
func TestResolve_FourLevelNestedChain(t *testing.T) {
	idD := "mem://d.extension.yaml"
	refD := ExtensionReference{ID: idD, Version: "v1alpha1"}
	docs := map[ExtensionReference][]byte{
		refA: []byte(`
metadata: {id: ` + idA + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idB + `, version: v1alpha1}]
  kinds: [{name: kindA}]
`),
		refB: []byte(`
metadata: {id: ` + idB + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idC + `, version: v1alpha1}]
  kinds: [{name: kindB}]
`),
		refC: []byte(`
metadata: {id: ` + idC + `, version: v1alpha1}
spec:
  dependencies: [{id: ` + idD + `, version: v1alpha1}]
  kinds: [{name: kindC}]
`),
		refD: []byte(`
metadata: {id: ` + idD + `, version: v1alpha1}
spec:
  kinds: [{name: kindD}]
`),
	}

	g := mustResolve(t, docs, refA)
	if len(g.Extensions) != 4 {
		t.Fatalf("expected 4 extensions, got %d: %v", len(g.Extensions), g.Extensions)
	}

	order := map[ExtensionReference]int{}
	for i, ext := range g.Extensions {
		order[ext.Metadata.Reference()] = i
	}
	if !(order[refD] < order[refC] && order[refC] < order[refB] && order[refB] < order[refA]) {
		t.Fatalf("expected topological order d,c,b,a; got order %v", order)
	}

	c, err := Merge(g)
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}
	for _, kind := range []string{"kindA", "kindB", "kindC", "kindD"} {
		if !c.IsValidKind(kind) {
			t.Errorf("expected kind %q from the resolved chain to be valid", kind)
		}
	}
}
