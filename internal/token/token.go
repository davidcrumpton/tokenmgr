// Package token builds, signs, and verifies self-describing bearer tokens.
// These are JWT-shaped (header.payload.signature, base64url, EdDSA) but the
// tool makes no claim about server-side auth semantics -- signing here is
// purely for provenance: "did tokenmgr issue this, and what does it say."
package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tokenmgr/internal/keysource"
)

type header struct {
	Alg string `json:"alg"` // always "EdDSA"
	Typ string `json:"typ"` // always "JWT"
}

// Claims is an ordered-ish bag of claim values. Registered claims (iss, sub,
// aud, iat, exp, jti) live alongside private claims in the same map --
// storage doesn't distinguish them, only the CLI/schema layer does.
type Claims map[string]interface{}

func b64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// Issue builds a signed token from the given claims using ks to sign.
// Callers are expected to have already populated iss/iat/jti as needed;
// Issue does not silently inject values so the schema layer stays in
// control of what appears in the payload.
func Issue(ks keysource.KeySource, claims Claims) (string, error) {
	h := header{Alg: "EdDSA", Typ: "JWT"}
	headerJSON, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("marshaling header: %w", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshaling claims: %w", err)
	}

	signingInput := b64(headerJSON) + "." + b64(payloadJSON)
	sig, err := ks.Sign([]byte(signingInput))
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signingInput + "." + b64(sig), nil
}

// VerifyResult is the outcome of checking a token against a public key.
type VerifyResult struct {
	Valid  bool
	Claims Claims
	Header struct {
		Alg string
		Typ string
	}
	Err error // set when Valid is false due to a structural problem, not just a bad signature
}

// Verify checks the token's signature against pub and, if valid, decodes
// the payload. It does not check exp/nbf -- lifecycle is left entirely to
// the holder of the token, per design (this tool does no revocation).
func Verify(pub ed25519.PublicKey, tok string) VerifyResult {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return VerifyResult{Valid: false, Err: fmt.Errorf("token must have 3 parts, got %d", len(parts))}
	}

	headerJSON, err := b64Decode(parts[0])
	if err != nil {
		return VerifyResult{Valid: false, Err: fmt.Errorf("decoding header: %w", err)}
	}
	payloadJSON, err := b64Decode(parts[1])
	if err != nil {
		return VerifyResult{Valid: false, Err: fmt.Errorf("decoding payload: %w", err)}
	}
	sig, err := b64Decode(parts[2])
	if err != nil {
		return VerifyResult{Valid: false, Err: fmt.Errorf("decoding signature: %w", err)}
	}

	var h header
	if err := json.Unmarshal(headerJSON, &h); err != nil {
		return VerifyResult{Valid: false, Err: fmt.Errorf("parsing header: %w", err)}
	}
	if h.Alg != "EdDSA" {
		return VerifyResult{Valid: false, Err: fmt.Errorf("unsupported alg %q", h.Alg)}
	}

	signingInput := parts[0] + "." + parts[1]
	sigValid := ed25519.Verify(pub, []byte(signingInput), sig)

	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return VerifyResult{Valid: false, Err: fmt.Errorf("parsing claims: %w", err)}
	}

	res := VerifyResult{Valid: sigValid, Claims: claims}
	res.Header.Alg = h.Alg
	res.Header.Typ = h.Typ
	if !sigValid {
		res.Err = fmt.Errorf("signature does not match")
	}
	return res
}

// NowUnix is a small seam so callers (and tests) don't call time.Now directly
// scattered across the codebase.
func NowUnix() int64 {
	return time.Now().Unix()
}
