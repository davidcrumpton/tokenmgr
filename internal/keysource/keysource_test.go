package keysource

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileKeySource(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "tokenmgr.key")
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(priv)
	_ = os.WriteFile(keyPath, []byte(encoded), 0600)

	s := &FileKeySource{Path: keyPath}
	pubkey, err := s.PublicKey()
	if err != nil {
		t.Errorf("failed to read key: %v", err)
	}
	if pubkey == nil {
		t.Errorf("expected non-nil public key, got nil")
	}

	// Test signing
	payload := []byte("test payload")
	signature, err := s.Sign(payload)
	if err != nil {
		t.Errorf("failed to sign: %v", err)
	}
	if signature == nil {
		t.Errorf("expected non-nil signature, got nil")
	}

	// Verify signature
	if !ed25519.Verify(pubkey, payload, signature) {
		t.Error("signature verification failed")
	}

	// Test Name()
	name := s.Name()
	if !strings.HasPrefix(name, "file:") {
		t.Errorf("expected name to start with 'file:', got %s", name)
	}
}

func TestFileKeySourceError(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "nonexistent.key")

	s := &FileKeySource{Path: keyPath}
	_, err := s.PublicKey()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	_, err = s.Sign([]byte("test"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFileKeySourceInvalidKey(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "invalid.key")
	// Write invalid key data
	_ = os.WriteFile(keyPath, []byte("invalid key data"), 0600)

	s := &FileKeySource{Path: keyPath}
	_, err := s.PublicKey()
	if err == nil {
		t.Error("expected error for invalid key file")
	}

	_, err = s.Sign([]byte("test"))
	if err == nil {
		t.Error("expected error for invalid key file")
	}
}

func TestGenerateKeyFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "tokenmgr.key")
	pubkey, err := GenerateKeyFile(keyPath)
	if err != nil {
		t.Errorf("failed to generate key: %v", err)
	}
	if pubkey == nil {
		t.Errorf("expected non-nil public key, got nil")
	}

	// verify file exists and has content
	_, err = os.Stat(keyPath)
	if err != nil {
		t.Errorf("expected key file to exist: %v", err)
	}

	// verify file content is valid base64
	content, err := os.ReadFile(keyPath)
	if err != nil {
		t.Errorf("failed to read key file: %v", err)
	}
	_, err = base64.StdEncoding.DecodeString(string(content))
	if err != nil {
		t.Errorf("expected key file content to be valid base64: %v", err)
	}
}

func TestGenerateKeyFileEmptyPath(t *testing.T) {
	_, err := GenerateKeyFile("")
	if err == nil {
		t.Errorf("expected error for empty key path")
	}
}

func TestGenerateKeyFileInvalidPath(t *testing.T) {
	// A directory cannot be used as a file path on any runner.
	_, err := GenerateKeyFile(t.TempDir())
	if err == nil {
		t.Errorf("expected error for invalid path")
	}
}

func TestEnvKeySource(t *testing.T) {
	// Set up test environment variable
	testKey := "test-key-value"
	os.Setenv("TEST_KEY", testKey)
	defer os.Unsetenv("TEST_KEY")

	s := &EnvKeySource{VarName: "TEST_KEY"}
	pubkey, err := s.PublicKey()
	if err == nil {
		t.Error("expected error for invalid key in env var")
	}

	// Test with valid key
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(priv)
	os.Setenv("TEST_KEY", encoded)
	defer os.Unsetenv("TEST_KEY")

	s = &EnvKeySource{VarName: "TEST_KEY"}
	pubkey, err = s.PublicKey()
	if err != nil {
		t.Errorf("failed to read key from env: %v", err)
	}
	if pubkey == nil {
		t.Errorf("expected non-nil public key, got nil")
	}

	// Test signing
	payload := []byte("test payload")
	signature, err := s.Sign(payload)
	if err != nil {
		t.Errorf("failed to sign: %v", err)
	}
	if signature == nil {
		t.Errorf("expected non-nil signature, got nil")
	}

	// Verify signature
	if !ed25519.Verify(pubkey, payload, signature) {
		t.Error("signature verification failed")
	}

	// Test Name()
	name := s.Name()
	if !strings.HasPrefix(name, "env:") {
		t.Errorf("expected name to start with 'env:', got %s", name)
	}
}

func TestEnvKeySourceError(t *testing.T) {
	// Test with unset environment variable
	s := &EnvKeySource{VarName: "NONEXISTENT_VAR"}
	_, err := s.PublicKey()
	if err == nil {
		t.Error("expected error for unset env var")
	}

	_, err = s.Sign([]byte("test"))
	if err == nil {
		t.Error("expected error for unset env var")
	}

	// Test with empty environment variable
	os.Setenv("EMPTY_KEY", "")
	defer os.Unsetenv("EMPTY_KEY")

	s = &EnvKeySource{VarName: "EMPTY_KEY"}
	_, err = s.PublicKey()
	if err == nil {
		t.Error("expected error for empty env var")
	}
}

func TestVaultKeySource(t *testing.T) {
	// Test that VaultKeySource is not yet implemented
	v := &VaultKeySource{
		Addr:    "https://vault.example.com",
		KeyName: "tokenmgr",
		Token:   "test-token",
	}

	_, err := v.Sign([]byte("test"))
	if err == nil {
		t.Error("expected error for unimplemented vault key source")
	}

	_, err = v.PublicKey()
	if err == nil {
		t.Error("expected error for unimplemented vault key source")
	}

	// Test Name()
	name := v.Name()
	expected := "vault:https://vault.example.com/tokenmgr"
	if name != expected {
		t.Errorf("expected name %s, got %s", expected, name)
	}
}

