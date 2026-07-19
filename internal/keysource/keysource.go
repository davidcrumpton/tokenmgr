// Package keysource abstracts where the Ed25519 signing key comes from.
// A KeySource never needs to expose the private key material to callers --
// it only signs and reports the public key.
package keysource

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

// KeySource signs payloads and reports the corresponding public key.
// Implementations decide where the private key actually lives.
type KeySource interface {
	// Sign returns a raw Ed25519 signature over payload.
	Sign(payload []byte) ([]byte, error)
	// PublicKey returns the raw 32-byte Ed25519 public key.
	PublicKey() (ed25519.PublicKey, error)
	// Name identifies the key source for display (e.g. "file:/path", "vault:transit/tokenmgr").
	Name() string
}

// FileKeySource loads an Ed25519 private key from a raw 64-byte seed file
// (as produced by `tokenmgr keygen`).
type FileKeySource struct {
	Path string
}

func (f *FileKeySource) load() (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, fmt.Errorf("reading key file %s: %w", f.Path, err)
	}
	raw, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("decoding key file %s: %w", f.Path, err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("key file %s: expected %d bytes, got %d", f.Path, ed25519.PrivateKeySize, len(raw))
	}
	return ed25519.PrivateKey(raw), nil
}

func (f *FileKeySource) Sign(payload []byte) ([]byte, error) {
	priv, err := f.load()
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(priv, payload), nil
}

func (f *FileKeySource) PublicKey() (ed25519.PublicKey, error) {
	priv, err := f.load()
	if err != nil {
		return nil, err
	}
	return priv.Public().(ed25519.PublicKey), nil
}

func (f *FileKeySource) Name() string {
	return "file:" + f.Path
}

// EnvKeySource loads a base64-encoded 64-byte Ed25519 seed from an
// environment variable. Useful for CI or ephemeral containers.
type EnvKeySource struct {
	VarName string
}

func (e *EnvKeySource) load() (ed25519.PrivateKey, error) {
	val := os.Getenv(e.VarName)
	if val == "" {
		return nil, fmt.Errorf("environment variable %s is not set", e.VarName)
	}
	raw, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", e.VarName, err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("%s: expected %d bytes, got %d", e.VarName, ed25519.PrivateKeySize, len(raw))
	}
	return ed25519.PrivateKey(raw), nil
}

func (e *EnvKeySource) Sign(payload []byte) ([]byte, error) {
	priv, err := e.load()
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(priv, payload), nil
}

func (e *EnvKeySource) PublicKey() (ed25519.PublicKey, error) {
	priv, err := e.load()
	if err != nil {
		return nil, err
	}
	return priv.Public().(ed25519.PublicKey), nil
}

func (e *EnvKeySource) Name() string {
	return "env:" + e.VarName
}

// VaultKeySource is a stub for signing via Hashicorp Vault's transit engine.
// The private key never leaves Vault; Sign() calls the transit/sign endpoint
// and PublicKey() calls transit/keys. Left unimplemented pending an HTTP
// client and Vault token/auth wiring -- deliberately not stdlib-only.
type VaultKeySource struct {
	Addr    string // e.g. "https://vault.crumpton.org"
	KeyName string // transit key name, e.g. "tokenmgr"
	Token   string // Vault token (or pulled from VAULT_TOKEN)
}

func (v *VaultKeySource) Sign(payload []byte) ([]byte, error) {
	return nil, fmt.Errorf("vault key source not yet implemented")
}

func (v *VaultKeySource) PublicKey() (ed25519.PublicKey, error) {
	return nil, fmt.Errorf("vault key source not yet implemented")
}

func (v *VaultKeySource) Name() string {
	return fmt.Sprintf("vault:%s/%s", v.Addr, v.KeyName)
}

// GenerateKeyFile creates a new Ed25519 keypair and writes the base64-encoded
// 64-byte seed to path. Returns the public key for display.
func GenerateKeyFile(path string) (ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(priv)
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("writing key file %s: %w", path, err)
	}
	return pub, nil
}
