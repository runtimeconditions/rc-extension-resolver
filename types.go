package resolver

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ExtensionReference identifies one exact release. IDs SHOULD use a resolver
// format, but URI syntax is not required.
type ExtensionReference struct {
	ID      string `yaml:"id" json:"id"`
	Version string `yaml:"version" json:"version"`
}

func (r ExtensionReference) Valid() bool { return r.ID != "" && r.Version != "" }

func (r ExtensionReference) String() string {
	return fmt.Sprintf("{id: %q, version: %q}", r.ID, r.Version)
}

func referenceFromFields(fields map[string]any) (ExtensionReference, error) {
	id, idOK := fields["id"].(string)
	version, versionOK := fields["version"].(string)
	if !idOK || !versionOK || id == "" || version == "" {
		return ExtensionReference{}, fmt.Errorf("extension reference requires non-empty string id and version")
	}
	return ExtensionReference{ID: id, Version: version}, nil
}

func (r *ExtensionReference) UnmarshalYAML(node *yaml.Node) error {
	var fields map[string]any
	if err := node.Decode(&fields); err != nil || len(fields) != 2 {
		return fmt.Errorf("extension reference requires exactly id and version")
	}
	reference, err := referenceFromFields(fields)
	if err != nil {
		return err
	}
	*r = reference
	return nil
}

type ExtensionDefinition struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       string            `yaml:"kind" json:"kind"`
	Metadata   ExtensionMetadata `yaml:"metadata" json:"metadata"`
	Spec       ExtensionSpec     `yaml:"spec" json:"spec"`
}

type ExtensionMetadata struct {
	ID      string `yaml:"id" json:"id"`
	Version string `yaml:"version" json:"version"`
}

func (m ExtensionMetadata) Reference() ExtensionReference {
	return ExtensionReference{ID: m.ID, Version: m.Version}
}

func (m *ExtensionMetadata) UnmarshalYAML(node *yaml.Node) error {
	var fields map[string]any
	if err := node.Decode(&fields); err != nil {
		return err
	}
	if _, hasURI := fields["uri"]; hasURI {
		return fmt.Errorf("metadata.uri is unsupported; use metadata.id and metadata.version")
	}
	reference, err := referenceFromFields(fields)
	if err != nil {
		return err
	}
	m.ID, m.Version = reference.ID, reference.Version
	return nil
}

type ExtensionSpec struct {
	Dependencies []ExtensionReference `yaml:"dependencies" json:"dependencies"`

	Kinds          []KindDef          `yaml:"kinds" json:"kinds"`
	InterfaceTypes []InterfaceTypeDef `yaml:"interfaceTypes" json:"interfaceTypes"`
	Schemas        []SchemaDef        `yaml:"schemas" json:"schemas"`
}

type KindDef struct {
	Name string `yaml:"name" json:"name"`
}

type InterfaceTypeDef struct {
	Name       string `yaml:"name" json:"name"`
	TargetKind string `yaml:"targetKind" json:"targetKind"`
}

type SchemaDef struct {
	ID                     string         `yaml:"id" json:"id"`
	AppliesToKind          string         `yaml:"appliesToKind" json:"appliesToKind"`
	AppliesToInterfaceType string         `yaml:"appliesToInterfaceType" json:"appliesToInterfaceType"`
	Description            string         `yaml:"description" json:"description"`
	Schema                 map[string]any `yaml:"schema" json:"schema"`
}
