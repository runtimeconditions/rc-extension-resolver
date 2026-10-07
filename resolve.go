package resolver

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// visitState is the three-color DFS scheme: visiting-but-not-visited
// means we've looped back to an ancestor, i.e. a cycle.
type visitState int

const (
	unvisited visitState = iota
	visiting
	visited
)

type Resolver struct {
	Loader LoaderFunc
}

func NewResolver(loader LoaderFunc) *Resolver {
	return &Resolver{Loader: loader}
}

// ResolvedGraph is every extension reachable from a root, deduplicated by
// exact (id, version) pair, in dependency-first order.
type ResolvedGraph struct {
	Extensions []*ExtensionDefinition
	ByID       map[ExtensionReference]*ExtensionDefinition
}

func (r *Resolver) Resolve(root ExtensionReference) (*ResolvedGraph, error) {
	g := &ResolvedGraph{ByID: make(map[ExtensionReference]*ExtensionDefinition)}
	state := make(map[ExtensionReference]visitState)

	var visit func(ExtensionReference, []ExtensionReference) error
	visit = func(reference ExtensionReference, path []ExtensionReference) error {
		if !reference.Valid() {
			return fmt.Errorf("extension reference requires non-empty string id and version")
		}
		switch state[reference] {
		case visiting:
			return &CycleError{Path: append(append([]ExtensionReference{}, path...), reference)}
		case visited:
			return nil
		}
		state[reference] = visiting

		ext, err := r.fetch(reference)
		if err != nil {
			return err
		}
		if ext.Metadata.Reference() != reference {
			return fmt.Errorf("fetch %s: document declares %s, expected the requested id and version", reference, ext.Metadata.Reference())
		}

		nextPath := append(append([]ExtensionReference{}, path...), reference)
		for _, dep := range ext.Spec.Dependencies {
			if err := visit(dep, nextPath); err != nil {
				return err
			}
		}

		state[reference] = visited
		g.ByID[reference] = ext
		g.Extensions = append(g.Extensions, ext)
		return nil
	}

	if err := visit(root, nil); err != nil {
		return nil, err
	}
	return g, nil
}

func (r *Resolver) fetch(reference ExtensionReference) (*ExtensionDefinition, error) {
	body, err := r.Loader(reference)
	if err != nil {
		return nil, &FetchError{Reference: reference, Err: err}
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, &FetchError{Reference: reference, Err: err}
	}

	var ext ExtensionDefinition
	if err := yaml.Unmarshal(data, &ext); err != nil {
		return nil, &FetchError{Reference: reference, Err: fmt.Errorf("parsing document: %w", err)}
	}
	if !ext.Metadata.Reference().Valid() {
		return nil, &FetchError{Reference: reference, Err: fmt.Errorf("metadata.id and metadata.version are required non-empty strings")}
	}
	return &ext, nil
}
