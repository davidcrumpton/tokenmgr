#!/bin/sh
# End-to-end check of the built binary: keygen -> issue -> verify, plus the
# properties that actually matter (tampering is detected, expiry is advisory,
# schema validation rejects bad claims). Run by both GitHub Actions and the
# GitLab pipeline so the two cannot drift apart.
#
# Usage: scripts/integration-test.sh   (from the repo root, or any directory)
set -eu

cd "$(dirname "$0")/.."

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT INT TERM

bin="$tmpdir/tokenmgr"
go build -o "$bin" ./cmd/tokenmgr

fail() {
    printf 'FAIL: %s\n' "$1" >&2
    exit 1
}

# assert_contains <haystack> <needle> <description>
assert_contains() {
    case "$1" in
    *"$2"*) ;;
    *) fail "$3 (expected to find: $2)" ;;
    esac
}

printf '== version ==\n'
"$bin" version | tee "$tmpdir/version.txt"
assert_contains "$(cat "$tmpdir/version.txt")" "tokenmgr " "version output"

printf '\n== keygen / pubkey ==\n'
keyfile="$tmpdir/tokenmgr.key"
"$bin" keygen --out "$keyfile"
[ -s "$keyfile" ] || fail "keygen did not write a key file"
pubkey=$("$bin" pubkey --key "$keyfile")
[ -n "$pubkey" ] || fail "pubkey printed nothing"

printf '\n== issue / verify roundtrip ==\n'
token=$("$bin" issue --key "$keyfile" \
    --iss ci-test --sub tokenmgr --aud ci \
    --claim purpose=integration-test)
verification=$("$bin" verify --pubkey "$pubkey" "$token")
printf '%s\n' "$verification"
assert_contains "$verification" "signature valid" "roundtrip verify"
assert_contains "$verification" "integration-test" "custom claim survived"

printf '\n== tampering is detected ==\n'
# Flip one character of the payload segment and expect a non-zero exit.
header=${token%%.*}
rest=${token#*.}
payload=${rest%%.*}
sig=${rest#*.}
case "$payload" in
*'A') mutated="${payload%?}B" ;;
*) mutated="${payload%?}A" ;;
esac
tampered="$header.$mutated.$sig"
if "$bin" verify --pubkey "$pubkey" "$tampered" >"$tmpdir/tampered.out" 2>&1; then
    cat "$tmpdir/tampered.out" >&2
    fail "tampered token verified successfully"
fi
assert_contains "$(cat "$tmpdir/tampered.out")" "signature invalid" "tampered token rejected"
printf 'tampered token rejected (exit non-zero)\n'

printf '\n== wrong key is rejected ==\n'
otherkey="$tmpdir/other.key"
"$bin" keygen --out "$otherkey" >/dev/null
otherpub=$("$bin" pubkey --key "$otherkey")
if "$bin" verify --pubkey "$otherpub" "$token" >"$tmpdir/wrongkey.out" 2>&1; then
    fail "token verified under an unrelated key"
fi
printf 'unrelated key rejected (exit non-zero)\n'

printf '\n== exp is advisory, not enforced ==\n'
# A 1s token still *verifies* after it expires: tokenmgr reports the claim and
# leaves enforcement to the caller. That is the documented contract, so assert
# it rather than a failure.
short_token=$("$bin" issue --key "$keyfile" --iss ci-test --exp 1s)
sleep 2
expired=$("$bin" verify --pubkey "$pubkey" "$short_token")
printf '%s\n' "$expired"
assert_contains "$expired" "signature valid" "expired token still verifies"
expiration=$(printf '%s\n' "$expired" | awk '$1 == "exp" {print $2}')
[ -n "$expiration" ] || fail "exp claim missing from output"
[ "$expiration" -le "$(date +%s)" ] || fail "exp claim is not in the past"

printf '\n== schema validation ==\n'
schemadir="$tmpdir/schemas"
mkdir -p "$schemadir"
cat >"$schemadir/ci.yaml" <<'YAML'
kind: ClaimSchema
name: ci
fields:
  - name: environment
    type: enum
    values: [dev, prod]
    default: dev
  - name: owner
    type: string
    required: true
YAML
"$bin" schema list --schema-dir "$schemadir" | tee "$tmpdir/schemas.txt"
assert_contains "$(cat "$tmpdir/schemas.txt")" "ci" "schema list"

# Missing required claim must fail.
if "$bin" issue --key "$keyfile" --schema-dir "$schemadir" --schema ci \
    >"$tmpdir/missing.out" 2>&1; then
    fail "issue accepted a missing required claim"
fi
printf 'missing required claim rejected\n'

# Valid claims must succeed.
"$bin" issue --key "$keyfile" --schema-dir "$schemadir" --schema ci \
    --claim owner=ci --claim environment=prod >/dev/null
printf 'valid claims accepted\n'

# Enum violation must fail.
if "$bin" issue --key "$keyfile" --schema-dir "$schemadir" --schema ci \
    --claim owner=ci --claim environment=staging >"$tmpdir/enum.out" 2>&1; then
    fail "issue accepted an out-of-enum value"
fi
printf 'out-of-enum value rejected\n'

printf '\n== bundled example schemas ==\n'
"$bin" schema list --schema-dir ./schemas | tee "$tmpdir/bundled.txt"
assert_contains "$(cat "$tmpdir/bundled.txt")" "qdrant" "bundled qdrant schema"

printf '\nAll integration checks passed.\n'
