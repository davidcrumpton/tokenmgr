package token

import (
	"crypto/ed25519"
	"os"
	"testing"

	"tokenmgr/internal/keysource"
)

func TestIssue(t *testing.T) {
	// Create a temporary key file for testing
	tmpKeyFile, err := os.CreateTemp("", "test-key")
	if err != nil {
		t.Fatalf("Failed to create temp key file: %v", err)
	}
	defer os.Remove(tmpKeyFile.Name())

	// Generate a keypair and write it to the file
	pub, err := keysource.GenerateKeyFile(tmpKeyFile.Name())
	if err != nil {
		t.Fatalf("Failed to generate key file: %v", err)
	}

	// Create a FileKeySource using the temporary key file
	ks := &keysource.FileKeySource{Path: tmpKeyFile.Name()}

	claims := Claims{
		"iss": "test-issuer",
		"sub": "test-subject",
		"exp": NowUnix() + 3600, // 1 hour from now
		"aud": "test-audience",
		"jti": "test-jti",
	}

	token, err := Issue(ks, claims)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	// Verify the token using the public key
	result := Verify(pub, token)
	if !result.Valid {
		t.Errorf("Token verification failed: %v", result.Err)
	}

	// Check that claims match
	expectedClaims := map[string]interface{}{
		"iss": "test-issuer",
		"sub": "test-subject",
		"exp": float64(NowUnix() + 3600),
		"aud": "test-audience",
		"jti": "test-jti",
	}
	for k, v := range expectedClaims {
		if result.Claims[k] != v {
			t.Errorf("Claim %s: expected %v, got %v", k, v, result.Claims[k])
		}
	}

	// Test with invalid number of token parts
	badToken := "part1.part2"
	result = Verify(pub, badToken)
	if result.Valid {
		t.Error("Expected verification to fail for bad token")
	}
}

func TestVerify(t *testing.T) {
	// Create a temporary key file for testing
	tmpKeyFile, err := os.CreateTemp("", "test-key")
	if err != nil {
		t.Fatalf("Failed to create temp key file: %v", err)
	}
	defer os.Remove(tmpKeyFile.Name())

	// Generate a keypair and write it to the file
	pub, err := keysource.GenerateKeyFile(tmpKeyFile.Name())
	if err != nil {
		t.Fatalf("Failed to generate key file: %v", err)
	}

	// Create a FileKeySource using the temporary key file
	ks := &keysource.FileKeySource{Path: tmpKeyFile.Name()}

	claims := Claims{
		"iss": "test-issuer",
		"sub": "test-subject",
		"exp": NowUnix() + 3600,
		"aud": "test-audience",
		"jti": "test-jti",
	}

	token, err := Issue(ks, claims)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	// Verify the token using the public key
	result := Verify(pub, token)
	if !result.Valid {
		t.Errorf("Token verification failed: %v", result.Err)
	}

	// Test with a wrong public key (should fail signature check)
	wrongPub := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(wrongPub, pub)
	wrongPub[0] ^= 1 // Flip one bit

	result = Verify(wrongPub, token)
	if result.Valid {
		t.Error("Expected verification to fail with wrong public key")
	}

	// Test with malformed header
	badToken := "not-base64..signature"
	result = Verify(pub, badToken)
	if result.Valid {
		t.Error("Expected verification to fail for malformed header")
	}
}

func TestB64(t *testing.T) {
	original := []byte("hello world")
	encoded := b64(original)
	decoded, err := b64Decode(encoded)
	if err != nil {
		t.Fatalf("b64Decode failed: %v", err)
	}
	if string(decoded) != string(original) {
		t.Errorf("b64 encoding/decoding failed: expected %s, got %s", original, decoded)
	}
}