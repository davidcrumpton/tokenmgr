package keysource

// func TestFileKeySource(t *testing.T) {
// 	tempDir := t.TempDir()
// 	keyPath := filepath.Join(tempDir, "tokenmgr.key")
// 	priv, _ := ed25519.GenerateKey(rand.Reader)
// 	encoded := base64.StdEncoding.EncodeToString(priv)
// 	_ = os.WriteFile(keyPath, []byte(encoded), 0600)

// 	s := NewFileSource(keyPath)
// 	_, err := s.PrivateKey()
// 	if err != nil {
// 		t.Errorf("failed to read key: %v", err)
// 	}
// }

// func TestEnvKeySource(t *testing.T) {
// 	priv, _ := ed25519.GenerateKey(rand.Reader)
// 	encoded := base64.StdEncoding.EncodeToString(priv)
// 	os.Setenv("TOKENMGR_KEY", encoded)

// 	s := NewEnvSource()
// 	_, err := s.PrivateKey()
// 	if err != nil {
// 		t.Errorf("failed to read key: %v", err)
// 	}
// 	os.Unsetenv("TOKENMGR_KEY")
// }

// func TestVaultKeySource(t *testing.T) {
// 	vaultAddr := os.Getenv("VAULT_ADDR")
// 	if vaultAddr == "" {
// 		t.Skip("VAULT_ADDR not set")
// 	}

// 	client, err := api.NewClient(api.DefaultConfig())
// 	if err != nil {
// 		t.Skipf("cannot create Vault client: %v", err)
// 	}

// 	s := NewVaultSource(client, "transit", "sign")
// 	_, err = s.PrivateKey()
// 	if err != nil {
// 		t.Errorf("failed to read key from Vault: %v", err)
// 	}
// }

// func TestCompositeKeySource(t *testing.T) {
// 	tempDir := t.TempDir()
// 	keyPath := filepath.Join(tempDir, "tokenmgr.key")
// 	priv, _ := ed25519.GenerateKey(rand.Reader)
// 	encoded := base64.StdEncoding.EncodeToString(priv)
// 	_ = os.WriteFile(keyPath, []byte(encoded), 0600)

// 	s := NewCompositeSource([]KeySource{NewFileSource(keyPath)})
// 	_, err := s.PrivateKey()
// 	if err != nil {
// 		t.Errorf("failed to read key: %v", err)
// 	}
// }

// func TestCompositeKeySourceMissing(t *testing.T) {
// 	s := NewCompositeSource([]KeySource{NewFileSource("nonexistent")})
// 	_, err := s.PrivateKey()
// 	if err == nil {
// 		t.Errorf("expected error for missing key")
// 	}
// }

// func TestEnvKeySourceNotSet(t *testing.T) {
// 	os.Unsetenv("TOKENMGR_KEY")

// 	s := NewEnvSource()
// 	_, err := s.PrivateKey()
// 	if err == nil {
// 		t.Errorf("expected error for missing key")
// 	}
// }

// func TestVaultKeySourceNoAddr(t *testing.T) {
// 	originalVaultAddr := os.Getenv("VAULT_ADDR")
// 	os.Unsetenv("VAULT_ADDR")

// 	defer os.Setenv("VAULT_ADDR", originalVaultAddr)

// 	s := NewVaultSource(nil, "transit", "sign")
// 	_, err := s.PrivateKey()
// 	if err == nil {
// 		t.Errorf("expected error for missing Vault address")
// 	}
// }

// func TestVaultKeySourceNoClient(t *testing.T) {
// 	s := NewVaultSource(nil, "transit", "sign")
// 	_, err := s.PrivateKey()
// 	if err == nil {
// 		t.Errorf("expected error for nil client")
// 	}
// }

// func TestCompositeKeySourceEmpty(t *testing.T) {
// 	s := NewCompositeSource([]KeySource{})
// 	_, err := s.PrivateKey()
// 	if err == nil {
// 		t.Errorf("expected error for empty composite source")
// 	}
// }

// func TestCompositeKeySourceAllFailed(t *testing.T) {
// 	s := NewCompositeSource([]KeySource{NewFileSource("nonexistent"), NewEnvSource(), NewVaultSource(nil, "transit", "sign")})
// 	_, err := s.PrivateKey()
// 	if err == nil {
// 		t.Errorf("expected error for all failed sources")
// 	}
// }

