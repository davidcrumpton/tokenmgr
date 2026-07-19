package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// func TestMain(t *testing.T) {
// 	t := &testing.T{}
// 	t.Run("exit code 1 when no command", func(t *testing.T) {
// 		testRunExitCode(t, t.Name(), 1)
// 	})

// 	t.Run("exit code 1 when unknown command", func(t *testing.T) {
// 		testRunExitCode(t, t.Name(), 1)
// 	})

// 	t.Run("exit code 0 when help", func(t *testing.T) {
// 		testRunExitCode(t, t.Name(), 0)
// 	})
// }

func testRunExitCode(t *testing.T, name string, wantExit int, args ...string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{name}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	gotExit := cmd.ProcessState.ExitCode()
	if gotExit != wantExit {
		t.Errorf("%s: exit code got %d, want %d\nstderr:\n%s", name, gotExit, wantExit, stderr.String())
	}
}

func TestMain(t *testing.T) {
	t.Run("keygen", testKeygen)
	t.Run("pubkey", testPubkey)
	t.Run("issue", testIssue)
	t.Run("verify", testVerify)
	t.Run("schema", testSchema)
}

func testKeygen(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "tokenmgr.key")

	testRunExitCode(t, "keygen", 0, "--out", outPath)

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("key file not created: %v", err)
	}

	// Check it looks like a key file (base64, 64 bytes)
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("cannot read key file: %v", err)
	}
	_, err = base64.StdEncoding.DecodeString(string(bytes.TrimSpace(data)))
	if err != nil {
		t.Errorf("key file not valid base64: %v", err)
	}
}
	
func testPubkey(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "tokenmgr.key")

	testRunExitCode(t, "keygen", 0, "--out", outPath)

	pub, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("cannot read key file: %v", err)
	}

	var stdout bytes.Buffer
	cmd := exec.Command(os.Args[0], "pubkey", "--key", outPath)
	cmd.Stdout = &stdout
	_ = cmd.Run()

	expected := base64.StdEncoding.EncodeToString(pub)
	if strings.TrimSpace(stdout.String()) != expected {
		t.Errorf("pubkey mismatch: got %q, want %q", stdout.String(), expected)
	}
}

func testIssue(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "tokenmgr.key")

	testRunExitCode(t, "keygen", 0, "--out", outPath)

	var tokenOut bytes.Buffer
	cmd := exec.Command(os.Args[0], "issue", "--key", outPath, "--claim", "role=admin", "--claim", "sub=123")
	cmd.Stdout = &tokenOut
	_ = cmd.Run()

	tokenStr := strings.TrimSpace(tokenOut.String())
	if tokenStr == "" {
		t.Fatal("no token issued")
	}
}

func testVerify(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "tokenmgr.key")

	testRunExitCode(t, "keygen", 0, "--out", outPath)

	var tokenOut bytes.Buffer
	cmd := exec.Command(os.Args[0], "issue", "--key", outPath, "--claim", "role=admin", "--claim", "sub=123")
	cmd.Stdout = &tokenOut
	_ = cmd.Run()

	tokenStr := strings.TrimSpace(tokenOut.String())
	if tokenStr == "" {
		t.Fatal("no token issued")
	}

	_, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("cannot read key file: %v", err)
	}

	var stdout bytes.Buffer
	cmd = exec.Command(os.Args[0], "verify", "--pubkey", outPath, tokenStr)
	cmd.Stdout = &stdout
	_ = cmd.Run()

	if !strings.Contains(stdout.String(), "signature valid") {
		t.Errorf("verification failed: %s", stdout.String())
	}
}

func testSchema(t *testing.T) {
	tempDir := t.TempDir()

	// Create a simple schema
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.Mkdir(schemaDir, 0755); err != nil {
		t.Fatalf("cannot create schema dir: %v", err)
	}
	schemaContent := []byte(`
type: claims
fields:
  email:
    type: string
    required: true
  role:
    type: string
    required: true
    oneOf:
      - admin
      - editor
      - viewer
`)
	if err := os.WriteFile(filepath.Join(schemaDir, "basic.yaml"), schemaContent, 0644); err != nil {
		t.Fatalf("cannot write schema: %v", err)
	}

	// Test listing schemas
	var listOut bytes.Buffer
	cmd := exec.Command(os.Args[0], "schema", "list", "--schema-dir", schemaDir)
	cmd.Stdout = &listOut
	_ = cmd.Run()

	if !strings.Contains(listOut.String(), "basic") {
		t.Errorf("schema list did not show basic: %s", listOut.String())
	}

	// Test showing schema
	var showOut bytes.Buffer
	cmd = exec.Command(os.Args[0], "schema", "show", "--schema-dir", schemaDir, "--schema", "basic")
	cmd.Stdout = &showOut
	_ = cmd.Run()

	if !strings.Contains(showOut.String(), "type: claims") {
		t.Errorf("schema show did not show schema content: %s", showOut.String())
	}
}