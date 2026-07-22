// tokenmgr issues and verifies self-describing bearer tokens. It is
// deliberately NOT an auth system: nothing is stored, nothing is revocable,
// and no server-side validation is assumed. The only thing tokenmgr can
// tell you is "did I sign this, and what does it say."
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"tokenmgr/internal/keysource"
	"tokenmgr/internal/schema"
	"tokenmgr/internal/token"
)

const envKeyName = "TOKENMGR_KEY"
const version = "v0.1.1"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "keygen":
		err = cmdKeygen(os.Args[2:])
	case "pubkey":
		err = cmdPubkey(os.Args[2:])
	case "issue":
		err = cmdIssue(os.Args[2:])
	case "verify":
		err = cmdVerify(os.Args[2:])
	case "schema":
		err = cmdSchema(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `tokenmgr `+version+` - Self-describing signed bearer tokens

Usage:
  tokenmgr keygen  --out <path>                       generate a new Ed25519 keypair
  tokenmgr pubkey  --key <path>                        print the public key for a key file
  tokenmgr issue   --key <path> [--schema-dir <dir> --schema <name>]
                   [--claim name=value ...] [--iss ...] [--sub ...]
                   [--aud ...] [--exp <duration|unix>]
  tokenmgr verify  --pubkey <base64-or-path> <token>
  tokenmgr schema  list --schema-dir <dir>
  tokenmgr schema  show --schema-dir <dir> --schema <name>
`)
}

// --- keygen ---

func cmdKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", "tokenmgr.key", "path to write the private key seed")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pub, err := keysource.GenerateKeyFile(*out)
	if err != nil {
		return err
	}
	fmt.Printf("wrote private key to %s\n", *out)
	fmt.Printf("public key: %s\n", base64.StdEncoding.EncodeToString(pub))
	return nil
}

// --- pubkey ---

func cmdPubkey(args []string) error {
	fs := flag.NewFlagSet("pubkey", flag.ExitOnError)
	keyPath := fs.String("key", "", "path to private key seed file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keyPath == "" {
		return fmt.Errorf("--key is required")
	}

	ks := &keysource.FileKeySource{Path: *keyPath}
	pub, err := ks.PublicKey()
	if err != nil {
		return err
	}
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	return nil
}

// --- issue ---

type claimFlags []string

func (c *claimFlags) String() string { return strings.Join(*c, ",") }
func (c *claimFlags) Set(v string) error {
	*c = append(*c, v)
	return nil
}

func cmdIssue(args []string) error {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	keyPath := fs.String("key", "", "path to private key seed file")
	schemaDir := fs.String("schema-dir", "", "directory containing claim schema YAML files")
	schemaName := fs.String("schema", "", "schema name to validate against (requires --schema-dir)")
	iss := fs.String("iss", "", "issuer claim")
	sub := fs.String("sub", "", "subject claim")
	aud := fs.String("aud", "", "audience claim")
	exp := fs.String("exp", "", "expiration: duration (e.g. 720h) or unix timestamp; omit for no expiry")
	var claims claimFlags
	fs.Var(&claims, "claim", "additional claim as name=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_, useEnvKey := os.LookupEnv(envKeyName)
	if !useEnvKey && *keyPath == "" {
		return fmt.Errorf("--key is required when " + envKeyName + " is undefined")
	}

	values := map[string]string{}
	for _, c := range claims {
		parts := strings.SplitN(c, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --claim %q, expected name=value", c)
		}
		values[parts[0]] = parts[1]
	}

	if *schemaName != "" {
		if *schemaDir == "" {
			return fmt.Errorf("--schema requires --schema-dir")
		}
		schemas, err := schema.LoadDir(*schemaDir)
		if err != nil {
			return err
		}
		s, ok := schemas[*schemaName]
		if !ok {
			return fmt.Errorf("no schema named %q in %s", *schemaName, *schemaDir)
		}
		if err := s.Validate(values); err != nil {
			return err
		}
	}

	payload := token.Claims{}
	for k, v := range values {
		payload[k] = v
	}
	if *iss != "" {
		payload["iss"] = *iss
	}
	if *sub != "" {
		payload["sub"] = *sub
	}
	if *aud != "" {
		payload["aud"] = *aud
	}
	payload["iat"] = token.NowUnix()
	payload["jti"] = newJTI()

	if *exp != "" {
		expUnix, err := resolveExp(*exp)
		if err != nil {
			return fmt.Errorf("--exp: %w", err)
		}
		payload["exp"] = expUnix
	}

	if !useEnvKey {
		ks := &keysource.FileKeySource{Path: *keyPath}
		tok, err := token.Issue(ks, payload)
		if err != nil {
			return err
		}
		fmt.Println(tok)
		return nil
	} else {
		ks := &keysource.EnvKeySource{VarName: envKeyName}
		tok, err := token.Issue(ks, payload)
		if err != nil {
			return err
		}
		fmt.Println(tok)
		return nil
	}
}

func newJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func resolveExp(v string) (int64, error) {
	if unix, err := strconv.ParseInt(v, 10, 64); err == nil {
		return unix, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("must be a unix timestamp or Go duration like 720h: %w", err)
	}
	return time.Now().Add(d).Unix(), nil
}

// --- verify ---

func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	pubkeyArg := fs.String("pubkey", "", "base64 public key, or path to a file containing one")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *pubkeyArg == "" {
		return fmt.Errorf("--pubkey is required")
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("expected exactly one token argument")
	}
	tok := fs.Arg(0)

	pub, err := loadPubkey(*pubkeyArg)
	if err != nil {
		return err
	}

	res := token.Verify(pub, tok)
	if !res.Valid {
		fmt.Println("✗ signature invalid")
		return res.Err
	}

	fmt.Println("✓ signature valid")
	printClaims(res.Claims)
	return nil
}

func loadPubkey(arg string) (ed25519.PublicKey, error) {
	raw := arg
	if data, err := os.ReadFile(arg); err == nil {
		raw = strings.TrimSpace(string(data))
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key: expected %d bytes, got %d", ed25519.PublicKeySize, len(decoded))
	}
	return ed25519.PublicKey(decoded), nil
}

func printClaims(c token.Claims) {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v, _ := json.Marshal(c[k])
		fmt.Printf("  %-12s %s\n", k, v)
	}
}

// --- schema ---

func cmdSchema(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: tokenmgr schema <list|show> --schema-dir <dir> [--schema <name>]")
	}
	sub := args[0]
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	dir := fs.String("schema-dir", "", "directory containing claim schema YAML files")
	name := fs.String("schema", "", "schema name (for 'show')")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *dir == "" {
		return fmt.Errorf("--schema-dir is required")
	}

	schemas, err := schema.LoadDir(*dir)
	if err != nil {
		return err
	}

	switch sub {
	case "list":
		names := make([]string, 0, len(schemas))
		for n := range schemas {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Println(n)
		}
	case "show":
		if *name == "" {
			return fmt.Errorf("--schema is required for 'show'")
		}
		s, ok := schemas[*name]
		if !ok {
			return fmt.Errorf("no schema named %q in %s", *name, *dir)
		}
		for _, f := range s.Fields {
			req := ""
			if f.Required {
				req = " (required)"
			}
			fmt.Printf("%-20s %-8s%s  %s\n", f.Name, f.Type, req, f.Description)
		}
	default:
		return fmt.Errorf("unknown schema subcommand %q", sub)
	}
	return nil
}
