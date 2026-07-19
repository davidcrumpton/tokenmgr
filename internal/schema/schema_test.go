package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListSchemas(t *testing.T) {
	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.Mkdir(schemaDir, 0755); err != nil {
		t.Fatalf("cannot create schema dir: %v", err)
	}
}

// 	// Create a dummy schema file
// 	schemaContent := []byte(`
// type: claims
// fields:
//   email:
//     type: string
//     required: true
//   role:
//     type: string
//     required: true
//     oneOf:
//       - admin
//       - editor
//       - viewer
// `)
// 	if err := os.WriteFile(filepath.Join(schemaDir, "basic.yaml"), schemaContent, 0644); err != nil {
// 		t.Fatalf("cannot write schema: %v", err)
// 	}

// 	// Mock the listSchemaNames to return our schema
// 	listSchemaNames = func(schemaDir string) ([]string, error) {
// 		if schemaDir != "" {
// 			return []string{"basic"}, nil
// 		}
// 		return nil, os.ErrNotExist
// 	}

// 	var out strings.Builder
// 	err := ListSchemas(&out, schemaDir)
// 	if err != nil {
// 		t.Errorf("ListSchemas error: %v", err)
// 	}

// 	result := out.String()
// 	if !strings.Contains(result, "basic") {
// 		t.Errorf("ListSchemas did not show basic schema: %s", result)
// 	}
// }

// func TestShowSchema(t *testing.T) {
// 	tempDir := t.TempDir()
// 	schemaDir := filepath.Join(tempDir, "schemas")
// 	if err := os.Mkdir(schemaDir, 0755); err != nil {
// 		t.Fatalf("cannot create schema dir: %v", err)
// 	}

// 	schemaContent := []byte(`
// type: claims
// fields:
//   email:
//     type: string
//     required: true
// `)
// 	if err := os.WriteFile(filepath.Join(schemaDir, "basic.yaml"), schemaContent, 0644); err != nil {
// 		t.Fatalf("cannot write schema: %v", err)
// 	}

// 	// Mock the loadSchema to return our schema
// 	var loadedSchema *Schema
// 	loadSchema = func(schemaDir, name string) (*Schema, error) {
// 		if schemaDir != "" && name == "basic" {
// 			loadedSchema = &Schema{
// 				Type: "claims",
// 				Fields: map[string]FieldSpec{
// 					"email": {
// 						Type:     "string",
// 						Required: true,
// 					},
// 				},
// 			}
// 			return loadedSchema, nil
// 		}
// 		return nil, os.ErrNotExist
// 	}

// 	var out strings.Builder
// 	err := ShowSchema(&out, schemaDir, "basic")
// 	if err != nil {
// 		t.Errorf("ShowSchema error: %v", err)
// 	}

// 	result := out.String()
// 	if !strings.Contains(result, "type: claims") {
// 		t.Errorf("ShowSchema did not show schema content: %s", result)
// 	}
// }

// func TestValidateSchema(t *testing.T) {
// 	tempDir := t.TempDir()
// 	schemaDir := filepath.Join(tempDir, "schemas")
// 	if err := os.Mkdir(schemaDir, 0755); err != nil {
// 		t.Fatalf("cannot create schema dir: %v", err)
// 	}

// 	schemaContent := []byte(`
// type: claims
// fields:
//   email:
//     type: string
//     required: true
//   age:
//     type: integer
//     minimum: 18
// `)
// 	if err := os.WriteFile(filepath.Join(schemaDir, "basic.yaml"), schemaContent, 0644); err != nil {
// 		t.Fatalf("cannot write schema: %v", err)
// 	}

// 	// Create a valid claims map
// 	claims := map[string]interface{}{
// 		"email": "[EMAIL_ADDRESS]",
// 		"age":   25,
// 	}

// 	var out strings.Builder
// 	err := ValidateSchema(&out, schemaDir, "basic", claims)
// 	if err != nil {
// 		t.Errorf("ValidateSchema error: %v", err)
// 	}

// 	result := out.String()
// 	if !strings.Contains(result, "validation successful") {
// 		t.Errorf("ValidateSchema validation failed unexpectedly: %s", result)
// 	}
// }

// func TestValidateSchemaMissingField(t *testing.T) {
// 	tempDir := t.TempDir()
// 	schemaDir := filepath.Join(tempDir, "schemas")
// 	if err := os.Mkdir(schemaDir, 0755); err != nil {
// 		t.Fatalf("cannot create schema dir: %v", err)
// 	}

// 	schemaContent := []byte(`
// type: claims
// fields:
//   email:
//     type: string
//     required: true
// `)
// 	if err := os.WriteFile(filepath.Join(schemaDir, "basic.yaml"), schemaContent, 0644); err != nil {
// 		t.Fatalf("cannot write schema: %v", err)
// 	}

// 	// Claims map missing required email field
// 	claims := map[string]interface{}{
// 		"age": 25,
// 	}

// 	var out strings.Builder
// 	err := ValidateSchema(&out, schemaDir, "basic", claims)
// 	if err == nil {
// 		t.Errorf("ValidateSchema expected error for missing required field but got nil")
// 	}

// 	result := out.String()
// 	if !strings.Contains(result, "email is required") {
// 		t.Errorf("ValidateSchema error message does not mention missing field: %s", result)
// 	}
// }

// func TestValidateSchemaInvalidValue(t *testing.T) {
// 	tempDir := t.TempDir()
// 	schemaDir := filepath.Join(tempDir, "schemas")
// 	if err := os.Mkdir(schemaDir, 0755); err != nil {
// 		t.Fatalf("cannot create schema dir: %v", err)
// 	}

// 	schemaContent := []byte(`
// type: claims
// fields:
//   role:
//     type: string
//     required: true
//     oneOf:
//       - admin
//       - editor
//       - viewer
// `)
// 	if err := os.WriteFile(filepath.Join(schemaDir, "basic.yaml"), schemaContent, 0644); err != nil {
// 		t.Fatalf("cannot write schema: %v", err)
// 	}

// 	// Claims map with invalid role value
// 	claims := map[string]interface{}{
// 		"email": "[EMAIL_ADDRESS]",
// 		"role":  "manager",
// 	}

// 	var out strings.Builder
// 	err := ValidateSchema(&out, schemaDir, "basic", claims)
// 	if err == nil {
// 		t.Errorf("ValidateSchema expected error for invalid value but got nil")
// 	}

// 	result := out.String()
// 	if !strings.Contains(result, "role must be one of: [admin editor viewer]") {
// 		t.Errorf("ValidateSchema error message does not mention invalid value: %s", result)
// 	}
// }


// func ValidateClaims(claims map[string]interface{}) error {
// 	return nil
// }
//
