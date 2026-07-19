package schema

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("loads valid schema", func(t *testing.T) {
		s, err := Load(filepath.Join("testdata", "valid.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "test", s.Name)
		assert.Len(t, s.Fields, 2)
	})

	t.Run("error on missing file", func(t *testing.T) {
		_, err := Load("nonexistent.yaml")
		assert.Error(t, err)
	})

	t.Run("error on invalid yaml", func(t *testing.T) {
		_, err := Load(filepath.Join("testdata", "invalid.yaml"))
		assert.Error(t, err)
	})

	t.Run("error on missing name", func(t *testing.T) {
		_, err := Load(filepath.Join("testdata", "missing-name.yaml"))
		assert.Error(t, err)
	})
}

func TestLoadDir(t *testing.T) {
	t.Run("loads all schemas in directory", func(t *testing.T) {
		schemas, err := LoadDir(filepath.Join("testdata", "schemas"))
		require.NoError(t, err)
		assert.Len(t, schemas, 2)
		assert.Contains(t, schemas, "test")
		assert.Contains(t, schemas, "another")
	})

	t.Run("error on duplicate schema name", func(t *testing.T) {
		_, err := LoadDir(filepath.Join("testdata", "duplicate"))
		assert.Error(t, err)
	})

	t.Run("ignores non-yaml files", func(t *testing.T) {
		schemas, err := LoadDir(filepath.Join("testdata", "mixed"))
		require.NoError(t, err)
		assert.Len(t, schemas, 1)
		assert.Contains(t, schemas, "test")
	})
}

func TestValidate(t *testing.T) {
	t.Run("validates required fields", func(t *testing.T) {
		s := &Schema{
			Name: "test",
			Fields: []Field{
				{Name: "name", Type: TypeString, Required: true},
			},
		}
		err := s.Validate(map[string]string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field \"name\" is required")
	})

	t.Run("validates enum fields", func(t *testing.T) {
		s := &Schema{
			Name: "test",
			Fields: []Field{
				{Name: "status", Type: TypeEnum, Values: []string{"active", "inactive"}, Required: true},
			},
		}
		err := s.Validate(map[string]string{"status": "invalid"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field \"status\": \"invalid\" is not one of")
	})

	t.Run("allows valid values", func(t *testing.T) {
		s := &Schema{
			Name: "test",
			Fields: []Field{
				{Name: "name", Type: TypeString, Required: true},
				{Name: "status", Type: TypeEnum, Values: []string{"active", "inactive"}},
			},
		}
		err := s.Validate(map[string]string{"name": "test", "status": "active"})
		assert.NoError(t, err)
	})
}