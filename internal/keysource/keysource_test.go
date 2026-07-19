package keysource

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
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

// func TestGenerateKeyFilePermissionError(t *testing.T) {
// 	tempDir := t.TempDir()
// 	keyPath := filepath.Join(tempDir, "tokenmgr.key")
// 	// Create directory with no write permissions
// 	_ = os.Mkdir(tempDir, 0555)
	
// 	_, err := GenerateKeyFile(keyPath)
// 	if err == nil {
// 		t.Errorf("expected error for permission denied")
// 	}
// }

func TestGenerateKeyFileInvalidPath(t *testing.T) {
	// Try to create key in root directory (should fail due to permissions)
	_, err := GenerateKeyFile("/tokenmgr.key")
	if err == nil {
		t.Errorf("expected error for invalid path")
	}
}

