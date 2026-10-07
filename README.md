# rc-extension-resolver

Pulls, caches, and validates Runtime Conditions extensions for downstream processing.

## What it does

A Runtime Conditions Profile lists the extensions it depends on and a set
of Conditions that use those extensions' kinds, interface types, and JSON
Schemas. This library does three things:

1. **Resolve** - fetch an extension and everything it depends on. Fails on
   cycles, broken references, or metadata that does not match the requested release.
2. **Merge** - combine every resolved extension into one `Catalog`. Fails
   if two extensions define the same interface type or schema.
3. **Validate** - check a Condition against the catalog: is its `kind`
   known, is its `interface.type` known, does it pass the bound JSON
   Schemas.

Extension authors are expected to namespace their `kind` names so they
don't collide with anyone else's. The library doesn't enforce that, so a
Condition can still name an `extension` field to pick between two
extensions that do collide on `kind` - but that's a fallback for a
naming mistake, not something to design around.

An extension release is identified by required, non-empty `id` and `version`
strings, compared exactly as a pair. `metadata.uri` is not an accepted alias.
IDs SHOULD use a resolver-supported format, such as an HTTPS or file URI or
an OCI reference; URI syntax is not required. Versions are exact strings,
not inferred from IDs, package versions, or version ranges.

`LoaderFunc` is `func(reference ExtensionReference) (io.ReadCloser, error)`.
HTTP loading retrieves `reference.ID`; custom loaders can use both fields to
locate a release through disk, OCI, or a configured mapping. The resolver
checks both metadata fields on the returned document. `NewInMemoryLoader`
and `ResolvedGraph.ByID` use `ExtensionReference` keys, so different versions
of the same ID remain distinct.

Profile declarations, extension dependencies, and Condition `extension`
selectors use objects containing exactly `id` and `version`:

```yaml
extensions:
  - id: example.widgets
    version: v1alpha1
```

## Usage

```go
loader := resolver.NewHTTPLoader(10 * time.Second)

reference := resolver.ExtensionReference{
    ID: "https://example.com/extensions/widgets.yaml",
    Version: "v1alpha1",
}
catalog, err := resolver.LoadAndMerge(loader, reference)
if err != nil {
    // resolution or merge failed
}

result, err := catalog.ValidateCondition(condition)
if err != nil {
    // condition references something the catalog doesn't know about
}
if !result.Valid {
    // result.Errors has the schema validation failures
}
```

## Install

```sh
go get github.com/runtimeconditions/rc-extension-resolver
```
