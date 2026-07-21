# tokenmgr

Self-describing signed bearer tokens for homelab infrastructure.

`tokenmgr` is **not an auth system**. It doesn't grant access, and services
you use it with (Qdrant, Vault, GitHub, whatever accepts an opaque bearer
string) never need to know it exists. All it does is answer one question:

> "Did I issue this token, and what did I say it was for?"

Tokens are JWT-shaped (`header.payload.signature`, base64url, signed with
Ed25519 / EdDSA) purely because that's a convenient, well-understood
container for a signed blob of JSON. There is no server-side validation,
no revocation, and no expiry enforcement — `exp` is just another claim you
can choose to check yourself if you want it.

## Why

Homelab services routinely accept arbitrary bearer tokens with no built-in
way to remember what a given token is *for*, who it belongs to, or when it
was created. `tokenmgr` signs a small JSON payload of your choosing so that
later — weeks or months on — you (or any tool you write) can decode it and
get a straight answer instead of an opaque string in a config file.

## Design

- **Nothing is stored.** No database, no manifest, no dashboard. The only
  thing that persists is your signing key. Everything else is derived
  per-call from the token itself.
- **Verification is stateless.** Given a token and a public key, `verify`
  decodes and checks it without needing to have seen it before.
- **Claims are unopinionated.** Registered JWT claims (`iss`, `sub`, `aud`,
  `iat`, `exp`, `jti`) and private claims (`service`, `owner`, `collection`,
  anything you want) live in the same flat map. Nothing is hardcoded.
- **Schemas are pure UX.** A YAML schema shapes what `issue` prompts for
  and validates presence/enum values, but has no bearing on storage or
  verification. You can issue tokens with zero schemas at all.

## Install

Requires Go 1.22+.

```bash
go build -o tokenmgr ./cmd/tokenmgr
```

## Usage

### 1. Generate a signing key

```bash
./tokenmgr keygen --out tokenmgr.key
# wrote private key to tokenmgr.key
# public key: P9hw69ClmtnxaQFX5Ftaa+nFx1Fj6e5CDYta1Ef4S+0=
```

Keep `tokenmgr.key` private. Share the printed public key (or run
`tokenmgr pubkey --key tokenmgr.key`) with anything that needs to verify
tokens but shouldn't be able to issue them.

### 2. (Optional) Define a claim schema

```yaml
# schemas/qdrant.yaml
kind: ClaimSchema
name: qdrant

fields:
  - name: service
    type: string
    required: true
    default: qdrant

  - name: collection
    type: string
    required: true
    description: Qdrant collection this token is associated with

  - name: environment
    type: enum
    values: [dev, test, prod]
    default: dev
```

Schema files can live anywhere — point `--schema-dir` at whatever directory
you keep them in.

```bash
./tokenmgr schema list --schema-dir ./schemas
./tokenmgr schema show --schema-dir ./schemas --schema qdrant
```

### 3. Issue a token

```bash
./tokenmgr issue --key tokenmgr.key \
  --schema-dir ./schemas --schema qdrant \
  --iss homelab-token-manager --sub qdrant --aud qdrant \
  --claim service=qdrant --claim collection=documents \
  --claim host=ai-server:6333 --claim purpose="Desktop Search" \
  --claim environment=prod --exp 720h
```

`iat` and `jti` are added automatically. `--exp` accepts either a Go
duration (`720h`) or a raw unix timestamp. Omit it for a token with no
expiration claim at all.

Validation against the schema (required fields, enum membership) happens
before signing; nothing is written anywhere.

### 4. Verify a token

```bash
./tokenmgr verify --pubkey P9hw69ClmtnxaQFX5Ftaa+nFx1Fj6e5CDYta1Ef4S+0= "$TOKEN"
```

```text
✓ signature valid
  aud          "qdrant"
  collection   "documents"
  environment  "prod"
  exp          1787022905
  host         "ai-server:6333"
  iat          1784430905
  iss          "homelab-token-manager"
  jti          "3d3f0cf3d4b18e37f5b4d1d91f2a8409"
  purpose      "Desktop Search"
  service      "qdrant"
  sub          "qdrant"
```

`--pubkey` accepts a raw base64 string or a path to a file containing one.
A tampered or wrongly-signed token fails with exit code 1:

```text
✗ signature invalid
  reason: signature does not match
```

## Key sources

Signing keys are abstracted behind a `KeySource` interface
(`internal/keysource`):

| Source | Status | Notes |
| --- | --- | --- |
| `FileKeySource` | done | base64-encoded 64-byte Ed25519 seed on disk |
| `EnvKeySource` | done | same format, read from an env var TOKENMGR_KEY |
| `VaultKeySource` | stub | intended to sign via Vault's transit engine so the private key never leaves Vault; not yet wired up |

``Note``

- `--key` is silently ignored if `TOKENMGR_KEY` is defined.
- Keys are always generated to a file on disk even if `TOKENMGR_KEY` is set.

## Project layout

```text
cmd/tokenmgr/main.go       CLI: keygen, pubkey, issue, verify, schema
internal/keysource/        KeySource interface + File/Env/Vault(stub)
internal/token/            JWT build, sign, verify (stdlib only, EdDSA)
internal/schema/           YAML claim schema loader + validation
schemas/qdrant.yaml        example schema
```

## Non-goals

- No authentication or authorization enforcement.
- No token storage, inventory, or dashboard.
- No revocation. Expiration, if set, is advisory only — nothing checks it
  but your own tooling if you choose to.
