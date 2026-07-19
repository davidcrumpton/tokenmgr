package token

import (
	"crypto/ed25519"
	"errors"
)

// func TestIssueAndVerify(t *testing.T) {
// 	pub, priv, err := ed25519.GenerateKey(rand.Reader)
// 	if err != nil {
// 		t.Fatalf("failed to generate key: %v", err)
// 	}

// 	ks := &testKeySource{priv: priv}

// 	claims := Claims{
// 		"sub": "test-subject",
// 		"aud": "test-audience",
// 		"exp": time.Now().Add(24 * time.Hour).Unix(),
// 	}

// 	tok, err := Issue(ks, claims)
// 	if err != nil {
// 		t.Fatalf("failed to issue token: %v", err)
// 	}

// 	result := Verify(pub, tok)
// 	if !result.Valid {
// 		t.Fatalf("token verification failed: %v", result.Err)
// 	}

// 	if !reflect.DeepEqual(result.Claims, claims) {
// 		t.Errorf("claims do not match. got %v, want %v", result.Claims, claims)
// 	}
// }

// Helper type for testing
type testKeySource struct {
	priv ed25519.PrivateKey
}

func (t *testKeySource) PrivateKey() (ed25519.PrivateKey, error) {
	return t.priv, nil
}

func (t *testKeySource) PublicKey() (ed25519.PublicKey, error) {
	return t.priv.Public().(ed25519.PublicKey), nil
}

func (t *testKeySource) Sign(data []byte) ([]byte, error) {
	return t.priv.Sign(nil, data, nil)
}

func (t *testKeySource) Verify(pub ed25519.PublicKey, data []byte, sig []byte) error {
	if ed25519.Verify(pub, data, sig) {
		return nil
	}
	return errors.New("signature verification failed")
}

