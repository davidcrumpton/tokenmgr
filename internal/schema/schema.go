// Package schema loads claim schemas -- CRD-style YAML files that describe
// which fields a given "kind" of token needs. The schema layer is pure UX:
// it shapes the `issue` command's prompts/flags. It has no bearing on
// storage (there isn't any) or on verification (tokens are self-contained).
package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// FieldType is the type hint used to validate/prompt for a field's value.
type FieldType string

const (
	TypeString FieldType = "string"
	TypeNumber FieldType = "number"
	TypeBool   FieldType = "boolean"
	TypeDate   FieldType = "date"
	TypeArray  FieldType = "array"
	TypeObject FieldType = "object"
	TypeEnum   FieldType = "enum"
)

// Field describes a single claim slot within a schema.
type Field struct {
	Name        string    `yaml:"name"`
	Type        FieldType `yaml:"type"`
	Required    bool      `yaml:"required"`
	Description string    `yaml:"description"`
	Values      []string  `yaml:"values"` // used when Type == TypeEnum
	Default     string    `yaml:"default"`
}

// Schema is a single claim-schema file, e.g. schemas/qdrant.yaml.
type Schema struct {
	Kind   string  `yaml:"kind"` // always "ClaimSchema" by convention, not enforced
	Name   string  `yaml:"name"` // e.g. "qdrant" -- selected via --schema
	Fields []Field `yaml:"fields"`
}

// Load parses a single schema file.
func Load(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading schema %s: %w", path, err)
	}
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing schema %s: %w", path, err)
	}
	if s.Name == "" {
		return nil, fmt.Errorf("schema %s: missing required 'name' field", path)
	}
	return &s, nil
}

// LoadDir scans dir for *.yaml / *.yml files and loads each as a Schema,
// keyed by Schema.Name. The directory location is entirely up to the user
// (flag, env var, or config) -- this function just reads what it's given.
func LoadDir(dir string) (map[string]*Schema, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading schema directory %s: %w", dir, err)
	}

	schemas := make(map[string]*Schema)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		s, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if _, exists := schemas[s.Name]; exists {
			return nil, fmt.Errorf("duplicate schema name %q (from %s)", s.Name, e.Name())
		}
		schemas[s.Name] = s
	}
	return schemas, nil
}

// Validate checks a set of provided values against the schema's field
// definitions. It only checks presence for required fields and enum
// membership -- it does not coerce types, since claim values are stored
// as whatever JSON type the caller provides.
func (s *Schema) Validate(values map[string]string) error {
	for _, f := range s.Fields {
		val, present := values[f.Name]
		if f.Required && !present {
			return fmt.Errorf("field %q is required by schema %q", f.Name, s.Name)
		}
		if f.Type == TypeEnum && present {
			ok := false
			for _, allowed := range f.Values {
				if val == allowed {
					ok = true
					break
				}
			}
			if !ok {
				return fmt.Errorf("field %q: %q is not one of %v", f.Name, val, f.Values)
			}
		}
	}
	return nil
}
