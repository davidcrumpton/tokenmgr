// Tests for main.go command-line interface
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crypto/ed25519"

	"tokenmgr/internal/keysource"
	"tokenmgr/internal/token"
)

// --- Test helpers ---

// captureOutput redirects stderr and stdout, runs fn, and returns captured output.
func captureOutput(fn func()) (string, string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	outBuf := new(bytes.Buffer)
	io.Copy(outBuf, rOut)
	rOut.Close()

	errBuf := new(bytes.Buffer)
	io.Copy(errBuf, rErr)
	rErr.Close()

	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String())
}

// createTestKeyFile generates a temporary key file and returns its path.
func createTestKeyFile(t *testing.T) string {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")

	pub, err := keysource.GenerateKeyFile(keyPath)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	_ = pub // unused, but validates key was generated
	return keyPath
}

// createTestToken creates a signed token for testing.
func createTestToken(t *testing.T, keyPath string, claims token.Claims) string {
	ks := &keysource.FileKeySource{Path: keyPath}
	tok, err := token.Issue(ks, claims)
	if err != nil {
		t.Fatalf("failed to create test token: %v", err)
	}
	return tok
}

// getPublicKeyFromFile extracts the public key from a key file.
func getPublicKeyFromFile(t *testing.T, keyPath string) ed25519.PublicKey {
	ks := &keysource.FileKeySource{Path: keyPath}
	pub, err := ks.PublicKey()
	if err != nil {
		t.Fatalf("failed to get public key: %v", err)
	}
	return pub
}

// --- cmdKeygen tests ---

func TestCmdKeygen_DefaultPath(t *testing.T) {
	dir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldCwd)

	err := cmdKeygen([]string{})
	if err != nil {
		t.Fatalf("cmdKeygen failed: %v", err)
	}

	// Check that the default file was created
	if _, err := os.Stat("tokenmgr.key"); os.IsNotExist(err) {
		t.Fatal("default key file not created")
	}
}

func TestCmdKeygen_CustomPath(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "custom.key")

	err := cmdKeygen([]string{"--out", keyPath})
	if err != nil {
		t.Fatalf("cmdKeygen failed: %v", err)
	}

	// Verify the file exists and contains valid base64
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("key file not readable: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		t.Fatalf("key file not valid base64: %v", err)
	}

	if len(decoded) != ed25519.PrivateKeySize {
		t.Fatalf("expected key size %d, got %d", ed25519.PrivateKeySize, len(decoded))
	}
}

func TestCmdKeygen_GeneratesValidKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test.key")

	err := cmdKeygen([]string{"--out", keyPath})
	if err != nil {
		t.Fatalf("cmdKeygen failed: %v", err)
	}

	// Verify we can use the key
	err = cmdPubkey([]string{"--key", keyPath})
	if err != nil {
		t.Fatalf("generated key is not usable: %v", err)
	}
}

// --- cmdPubkey tests ---

func TestCmdPubkey_Success(t *testing.T) {
	keyPath := createTestKeyFile(t)
	expected := base64.StdEncoding.EncodeToString(getPublicKeyFromFile(t, keyPath))

	stdout, stderr := captureOutput(func() {
		err := cmdPubkey([]string{"--key", keyPath})
		if err != nil {
			t.Errorf("cmdPubkey failed: %v", err)
		}
	})

	if !strings.Contains(stdout, expected) {
		t.Fatalf("expected %s in output, got: %s", expected, stdout)
	}

	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestCmdPubkey_MissingKeyFlag(t *testing.T) {
	err := cmdPubkey([]string{})
	if err == nil {
		t.Fatal("expected error when --key is missing")
	}
	if !strings.Contains(err.Error(), "--key is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCmdPubkey_InvalidKeyFile(t *testing.T) {
	err := cmdPubkey([]string{"--key", "/nonexistent/path/key"})
	if err == nil {
		t.Fatal("expected error for nonexistent key file")
	}
}

// --- claimFlags tests ---

func TestClaimFlags_String(t *testing.T) {
	var cf claimFlags
	cf = append(cf, "foo=bar", "baz=qux")
	result := cf.String()
	if result != "foo=bar,baz=qux" {
		t.Fatalf("expected 'foo=bar,baz=qux', got %q", result)
	}
}

func TestClaimFlags_Set(t *testing.T) {
	var cf claimFlags
	cf.Set("claim1=value1")
	cf.Set("claim2=value2")

	if len(cf) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(cf))
	}
	if cf[0] != "claim1=value1" {
		t.Fatalf("expected first claim to be 'claim1=value1', got %q", cf[0])
	}
}

// --- cmdIssue tests ---

func TestCmdIssue_MissingKeyFlag(t *testing.T) {
	err := cmdIssue([]string{})
	if err == nil {
		t.Fatal("expected error when --key is missing")
	}
	if !strings.Contains(err.Error(), "--key is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdIssue_BasicToken(t *testing.T) {
	keyPath := createTestKeyFile(t)

	stdout, _ := captureOutput(func() {
		err := cmdIssue([]string{"--key", keyPath})
		if err != nil {
			t.Errorf("cmdIssue failed: %v", err)
		}
	})

	tok := strings.TrimSpace(stdout)
	if !strings.Contains(tok, ".") {
		t.Fatalf("expected JWT format token, got: %s", tok)
	}

	// Verify the token is valid
	pub := getPublicKeyFromFile(t, keyPath)
	res := token.Verify(pub, tok)
	if !res.Valid {
		t.Fatalf("issued token failed verification: %v", res.Err)
	}

	// Check that required claims are present
	if _, ok := res.Claims["iat"]; !ok {
		t.Fatal("missing 'iat' claim")
	}
	if _, ok := res.Claims["jti"]; !ok {
		t.Fatal("missing 'jti' claim")
	}
}

func TestCmdIssue_WithRegisteredClaims(t *testing.T) {
	keyPath := createTestKeyFile(t)

	stdout, _ := captureOutput(func() {
		err := cmdIssue([]string{
			"--key", keyPath,
			"--iss", "test-issuer",
			"--sub", "test-subject",
			"--aud", "test-audience",
		})
		if err != nil {
			t.Errorf("cmdIssue failed: %v", err)
		}
	})

	tok := strings.TrimSpace(stdout)
	pub := getPublicKeyFromFile(t, keyPath)
	res := token.Verify(pub, tok)
	if !res.Valid {
		t.Fatalf("token verification failed: %v", res.Err)
	}

	if res.Claims["iss"] != "test-issuer" {
		t.Fatalf("iss claim mismatch: expected 'test-issuer', got %v", res.Claims["iss"])
	}
	if res.Claims["sub"] != "test-subject" {
		t.Fatalf("sub claim mismatch: expected 'test-subject', got %v", res.Claims["sub"])
	}
	if res.Claims["aud"] != "test-audience" {
		t.Fatalf("aud claim mismatch: expected 'test-audience', got %v", res.Claims["aud"])
	}
}

func TestCmdIssue_WithCustomClaims(t *testing.T) {
	keyPath := createTestKeyFile(t)

	stdout, _ := captureOutput(func() {
		err := cmdIssue([]string{
			"--key", keyPath,
			"--claim", "username=alice",
			"--claim", "role=admin",
		})
		if err != nil {
			t.Errorf("cmdIssue failed: %v", err)
		}
	})

	tok := strings.TrimSpace(stdout)
	pub := getPublicKeyFromFile(t, keyPath)
	res := token.Verify(pub, tok)
	if !res.Valid {
		t.Fatalf("token verification failed: %v", res.Err)
	}

	if res.Claims["username"] != "alice" {
		t.Fatalf("username claim mismatch: got %v", res.Claims["username"])
	}
	if res.Claims["role"] != "admin" {
		t.Fatalf("role claim mismatch: got %v", res.Claims["role"])
	}
}

func TestCmdIssue_InvalidClaimFormat(t *testing.T) {
	keyPath := createTestKeyFile(t)

	err := cmdIssue([]string{
		"--key", keyPath,
		"--claim", "invalid-no-equals",
	})

	if err == nil {
		t.Fatal("expected error for invalid claim format")
	}
	if !strings.Contains(err.Error(), "invalid --claim") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdIssue_WithExpiration_Duration(t *testing.T) {
	keyPath := createTestKeyFile(t)
	beforeIssue := time.Now().Unix()

	stdout, _ := captureOutput(func() {
		err := cmdIssue([]string{
			"--key", keyPath,
			"--exp", "24h",
		})
		if err != nil {
			t.Errorf("cmdIssue failed: %v", err)
		}
	})

	tok := strings.TrimSpace(stdout)
	pub := getPublicKeyFromFile(t, keyPath)
	res := token.Verify(pub, tok)
	if !res.Valid {
		t.Fatalf("token verification failed: %v", res.Err)
	}

	expVal, ok := res.Claims["exp"]
	if !ok {
		t.Fatal("missing 'exp' claim")
	}

	// Convert to int64 for comparison
	var expInt int64
	switch v := expVal.(type) {
	case float64:
		expInt = int64(v)
	case int64:
		expInt = v
	default:
		t.Fatalf("unexpected type for exp claim: %T", v)
	}

	expectedExp := beforeIssue + (24 * 3600)
	// Allow 5 second tolerance for test execution time
	if expInt < expectedExp-5 || expInt > expectedExp+5 {
		t.Fatalf("expected exp around %d, got %d", expectedExp, expInt)
	}
}

func TestCmdIssue_WithExpiration_UnixTimestamp(t *testing.T) {
	keyPath := createTestKeyFile(t)
	expTimestamp := time.Now().Add(72 * time.Hour).Unix()

	stdout, _ := captureOutput(func() {
		err := cmdIssue([]string{
			"--key", keyPath,
			"--exp", fmt.Sprintf("%d", expTimestamp),
		})
		if err != nil {
			t.Errorf("cmdIssue failed: %v", err)
		}
	})

	tok := strings.TrimSpace(stdout)
	pub := getPublicKeyFromFile(t, keyPath)
	res := token.Verify(pub, tok)
	if !res.Valid {
		t.Fatalf("token verification failed: %v", res.Err)
	}

	expVal, ok := res.Claims["exp"]
	if !ok {
		t.Fatal("missing 'exp' claim")
	}

	var expInt int64
	switch v := expVal.(type) {
	case float64:
		expInt = int64(v)
	case int64:
		expInt = v
	default:
		t.Fatalf("unexpected type for exp claim: %T", v)
	}

	if expInt != expTimestamp {
		t.Fatalf("expected exp %d, got %d", expTimestamp, expInt)
	}
}

func TestCmdIssue_InvalidExpiration(t *testing.T) {
	keyPath := createTestKeyFile(t)

	err := cmdIssue([]string{
		"--key", keyPath,
		"--exp", "invalid-duration",
	})

	if err == nil {
		t.Fatal("expected error for invalid expiration")
	}
	if !strings.Contains(err.Error(), "exp") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- newJTI tests ---

func TestNewJTI_GeneratesValidHex(t *testing.T) {
	jti := newJTI()
	if len(jti) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected JTI length 32, got %d", len(jti))
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(jti); err != nil {
		t.Fatalf("JTI is not valid hex: %v", err)
	}
}

func TestNewJTI_Uniqueness(t *testing.T) {
	jti1 := newJTI()
	jti2 := newJTI()
	if jti1 == jti2 {
		t.Fatal("JTI should be unique")
	}
}

// --- resolveExp tests ---

func TestResolveExp_UnixTimestamp(t *testing.T) {
	targetTime := int64(1893456000)
	result, err := resolveExp("1893456000")
	if err != nil {
		t.Fatalf("resolveExp failed: %v", err)
	}
	if result != targetTime {
		t.Fatalf("expected %d, got %d", targetTime, result)
	}
}

func TestResolveExp_Duration(t *testing.T) {
	before := time.Now().Unix()
	result, err := resolveExp("720h")
	if err != nil {
		t.Fatalf("resolveExp failed: %v", err)
	}

	// Result should be approximately 720 hours from now
	expected := before + (720 * 3600)
	if result < expected-1 || result > expected+1 {
		t.Fatalf("expected ~%d, got %d", expected, result)
	}
}

func TestResolveExp_InvalidInput(t *testing.T) {
	_, err := resolveExp("not-a-duration-or-timestamp")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
	if !strings.Contains(err.Error(), "must be a unix timestamp") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- cmdVerify tests ---

func TestCmdVerify_ValidToken(t *testing.T) {
	keyPath := createTestKeyFile(t)
	claims := token.Claims{
		"iss": "test",
		"sub": "user",
		"iat": time.Now().Unix(),
		"jti": newJTI(),
	}
	tok := createTestToken(t, keyPath, claims)
	pubKey := getPublicKeyFromFile(t, keyPath)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	stdout, _ := captureOutput(func() {
		err := cmdVerify([]string{
			"--pubkey", pubKeyB64,
			tok,
		})
		if err != nil {
			t.Errorf("cmdVerify failed: %v", err)
		}
	})

	if !strings.Contains(stdout, "signature valid") {
		t.Fatalf("expected 'signature valid' in output, got: %s", stdout)
	}
}

func TestCmdVerify_InvalidToken(t *testing.T) {
	keyPath := createTestKeyFile(t)
	pubKey := getPublicKeyFromFile(t, keyPath)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	// Create a token with a different key to make signature invalid
	otherKeyPath := createTestKeyFile(t)
	claims := token.Claims{
		"iss": "test",
		"iat": time.Now().Unix(),
		"jti": newJTI(),
	}
	tok := createTestToken(t, otherKeyPath, claims)

	stdout, _ := captureOutput(func() {
		cmdVerify([]string{
			"--pubkey", pubKeyB64,
			tok,
		})
	})

	if !strings.Contains(stdout, "signature invalid") {
		t.Fatalf("expected 'signature invalid' in output, got: %s", stdout)
	}
}

func TestCmdVerify_MissingPubkeyFlag(t *testing.T) {
	err := cmdVerify([]string{"sometoken"})
	if err == nil {
		t.Fatal("expected error when --pubkey is missing")
	}
	if !strings.Contains(err.Error(), "--pubkey is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdVerify_MissingTokenArgument(t *testing.T) {
	keyPath := createTestKeyFile(t)
	pubKey := getPublicKeyFromFile(t, keyPath)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	err := cmdVerify([]string{"--pubkey", pubKeyB64})
	if err == nil {
		t.Fatal("expected error when token argument is missing")
	}
	if !strings.Contains(err.Error(), "expected exactly one token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdVerify_PublicKeyFromFile(t *testing.T) {
	keyPath := createTestKeyFile(t)
	claims := token.Claims{
		"iss": "test",
		"iat": time.Now().Unix(),
		"jti": newJTI(),
	}
	tok := createTestToken(t, keyPath, claims)
	pubKey := getPublicKeyFromFile(t, keyPath)

	// Write public key to a file
	pubKeyFile := filepath.Join(t.TempDir(), "pubkey.txt")
	os.WriteFile(pubKeyFile, []byte(base64.StdEncoding.EncodeToString(pubKey)), 0644)

	stdout, _ := captureOutput(func() {
		err := cmdVerify([]string{
			"--pubkey", pubKeyFile,
			tok,
		})
		if err != nil {
			t.Errorf("cmdVerify failed: %v", err)
		}
	})

	if !strings.Contains(stdout, "signature valid") {
		t.Fatalf("expected 'signature valid', got: %s", stdout)
	}
}

func TestCmdVerify_InvalidPublicKey(t *testing.T) {
	err := cmdVerify([]string{
		"--pubkey", "not-valid-base64!!!",
		"sometoken",
	})
	if err == nil {
		t.Fatal("expected error for invalid public key")
	}
}

// --- loadPubkey tests ---

func TestLoadPubkey_Base64String(t *testing.T) {
	keyPath := createTestKeyFile(t)
	pubKey := getPublicKeyFromFile(t, keyPath)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	loaded, err := loadPubkey(pubKeyB64)
	if err != nil {
		t.Fatalf("loadPubkey failed: %v", err)
	}

	if !bytes.Equal(loaded, pubKey) {
		t.Fatal("loaded public key doesn't match")
	}
}

func TestLoadPubkey_FromFile(t *testing.T) {
	keyPath := createTestKeyFile(t)
	pubKey := getPublicKeyFromFile(t, keyPath)

	// Write public key to a file
	pubKeyFile := filepath.Join(t.TempDir(), "pubkey.txt")
	os.WriteFile(pubKeyFile, []byte(base64.StdEncoding.EncodeToString(pubKey)), 0644)

	loaded, err := loadPubkey(pubKeyFile)
	if err != nil {
		t.Fatalf("loadPubkey failed: %v", err)
	}

	if !bytes.Equal(loaded, pubKey) {
		t.Fatal("loaded public key doesn't match")
	}
}

func TestLoadPubkey_InvalidBase64(t *testing.T) {
	_, err := loadPubkey("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if !strings.Contains(err.Error(), "decoding public key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadPubkey_WrongKeySize(t *testing.T) {
	// Generate invalid size key
	invalidKey := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	_, err := loadPubkey(invalidKey)
	if err == nil {
		t.Fatal("expected error for wrong key size")
	}
	if !strings.Contains(err.Error(), "expected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- cmdSchema tests ---

func TestCmdSchema_MissingSchemaDir(t *testing.T) {
	err := cmdSchema([]string{"list"})
	if err == nil {
		t.Fatal("expected error when --schema-dir is missing")
	}
	if !strings.Contains(err.Error(), "--schema-dir is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdSchema_UnknownSubcommand(t *testing.T) {
	dir := t.TempDir()
	err := cmdSchema([]string{"invalid", "--schema-dir", dir})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown schema subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCmdSchema_NoSubcommand(t *testing.T) {
	err := cmdSchema([]string{})
	if err == nil {
		t.Fatal("expected error when no subcommand provided")
	}
}

func TestCmdSchema_ShowMissingSchema(t *testing.T) {
	dir := t.TempDir()
	err := cmdSchema([]string{"show", "--schema-dir", dir})
	if err == nil {
		t.Fatal("expected error when --schema is missing for show")
	}
	if !strings.Contains(err.Error(), "--schema is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- main function entry point tests ---

func TestMain_NoArgs(t *testing.T) {
	// Save original os.Args and os.Exit
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"tokenmgr"}

	// Can't easily test os.Exit in normal tests, so we just verify
	// the usage function doesn't panic
	_, _ = captureOutput(func() {
		if len(os.Args) < 2 {
			usage()
		}
	})
}

func TestMain_UnknownCommand(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"tokenmgr", "unknown"}

	_, stderr := captureOutput(func() {
		usage()
	})

	if !strings.Contains(stderr, "tokenmgr") {
		t.Fatalf("expected usage in stderr")
	}
}

func TestMain_HelpCommand(t *testing.T) {
	_, _ = captureOutput(func() {
		usage()
	})
	// Just verify it doesn't panic
}

// --- Integration-style tests ---

func TestIssueAndVerify_RoundTrip(t *testing.T) {
	// Generate a key
	keyPath := createTestKeyFile(t)
	pubKey := getPublicKeyFromFile(t, keyPath)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	// Issue a token
	var token string
	_, _ = captureOutput(func() {
		cmdIssue([]string{
			"--key", keyPath,
			"--iss", "test-issuer",
			"--claim", "user_id=12345",
		})
	})

	// Extract the token from output would require modifying the test,
	// so we'll just verify the core flow works by calling functions directly

	stdout, _ := captureOutput(func() {
		cmdIssue([]string{
			"--key", keyPath,
			"--iss", "test-issuer",
			"--claim", "user_id=12345",
		})
	})
	token = strings.TrimSpace(stdout)

	// Verify the token
	verifyOutput, _ := captureOutput(func() {
		cmdVerify([]string{
			"--pubkey", pubKeyB64,
			token,
		})
	})

	if !strings.Contains(verifyOutput, "signature valid") {
		t.Fatalf("token verification failed: %s", verifyOutput)
	}
	if !strings.Contains(verifyOutput, "test-issuer") {
		t.Fatalf("issuer claim not in output: %s", verifyOutput)
	}
}
