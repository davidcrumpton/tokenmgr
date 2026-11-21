# Packaging and releases

How a tag becomes an installable artifact. This mirrors the setup used by
[`mcwsd`](https://github.com/davidcrumpton/mcwsd) so the two projects release
the same way.

## What one tag produces

Pushing a `v*` tag to the GitHub remote triggers
`.github/workflows/release.yml`, which runs GoReleaser once and produces:

| Output | Destination |
| --- | --- |
| `tokenmgr_<ver>_<os>_<arch>.tar.gz` (`.zip` on Windows) | GitHub release |
| `.deb`, `.rpm`, `.apk` for amd64 + arm64 | GitHub release |
| `checksums.txt` + `checksums.txt.sigstore.json` | GitHub release |
| `*.spdx.json` SBOM per archive | GitHub release |
| Build provenance attestations | GitHub attestation store |
| `Formula/tokenmgr.rb` | `davidcrumpton/homebrew-tap` |
| `bucket/tokenmgr.json` | `davidcrumpton/scoop-bucket` |

Build matrix: `linux`, `darwin`, `freebsd`, `windows` × `amd64`, `arm64`, all
`CGO_ENABLED=0`.

There is deliberately **no container image**. `tokenmgr` is a local CLI that
operates on a token you already hold; an image would add a registry dependency
without enabling anything a tarball does not.

## Install paths

```sh
brew install davidcrumpton/tap/tokenmgr          # macOS / Linuxbrew
sudo apt install ./tokenmgr_<ver>_linux_amd64.deb
sudo dnf install ./tokenmgr-<ver>-1.x86_64.rpm
sudo apk add --allow-untrusted ./tokenmgr_<ver>_linux_amd64.apk
go install github.com/davidcrumpton/tokenmgr/cmd/tokenmgr@v<ver>
```

Archives contain `tokenmgr`, `README.md`, `LICENSE` and the example schemas
under `schemas/`. Packages additionally install the schemas to
`/usr/share/tokenmgr/schemas` and the docs to `/usr/share/doc/tokenmgr/`.

## Versioning

`cmd/tokenmgr/main.go` carries the source default:

```go
var version = "v0.1.1"
```

The release workflow refuses to publish if the tag does not equal that string,
so a locally built binary and a released binary never disagree. GoReleaser
re-stamps the same value at link time:

```
-X main.version=v{{ .Version }}    # goreleaser strips the leading v, we add it back
```

Two things this depends on, both easy to break:

1. **`version` must stay a `var`.** The linker's `-X` flag cannot assign to a
   `const`. With a `const`, the flag is silently ignored and every release
   reports the source default. This was the actual state of the tree before
   packaging and was fixed here; `-X main.version=…` on a `const` produces no
   error, just a wrong binary.
2. **The module path must match the repo.** `go install` resolves
   `github.com/davidcrumpton/tokenmgr/cmd/tokenmgr@v0.1.1` by checking the
   `module` line in `go.mod`; a bare `module tokenmgr` fails with
   *"module declares its path as: tokenmgr"*. Also fixed here.

## Cutting a release

```sh
git tag v0.1.2
git push github v0.1.2
```

Watch the Release workflow. Then confirm:

```sh
gh release view v0.1.2 --repo davidcrumpton/tokenmgr
brew update && brew upgrade davidcrumpton/tap/tokenmgr
tokenmgr version
```

## Local verification without tagging

The same snapshot command CI runs, which renders every archive, package,
formula and manifest without publishing anything:

```sh
goreleaser release --snapshot --clean --skip=publish,sbom,sign
```

Output lands in `dist/`. To exercise the generated formula for real, copy
`dist/homebrew/Formula/tokenmgr.rb` into a local tap with the `url` rewritten
to the local archive and run `brew install` + `brew test`. The formula's test
block does a full `keygen` → `issue` → `verify` roundtrip, so a broken binary
fails the install rather than the user's first command.

Note that `goreleaser check` exits non-zero on the deprecated `brews` property.
Formulae are still supported and are what Linuxbrew users need, so the real
gate is the snapshot release above, not `check`.

## One-time setup (already done)

These are recorded so they can be rebuilt if lost:

1. **Repositories** — `davidcrumpton/homebrew-tap` and
   `davidcrumpton/scoop-bucket` exist, both with `main` as the default branch
   and a `Formula/` and `bucket/` directory respectively.
2. **`TAP_GITHUB_TOKEN`** — a PAT with `contents:write` on both repos, stored
   as a repository secret on `tokenmgr`. GoReleaser cannot push to a second
   repository with the workflow's own `GITHUB_TOKEN`.
3. **GitHub remote** — the local repo has two remotes:

   ```sh
   origin   git@gitlab.crumpton.org:crumptonorg/tokenmgr.git
   github   git@github.com:davidcrumpton/tokenmgr.git
   ```

   Releases are cut from `github`. The GitLab remote keeps its own pipeline
   (`.gitlab-ci.yml`), which runs the same `scripts/integration-test.sh` as
   GitHub Actions.

## CI layout

| Workflow | Trigger | Jobs |
| --- | --- | --- |
| `ci.yml` | push to `main`, PRs | lint, test (ubuntu + macos), cross-build, snapshot release |
| `release.yml` | `v*` tag | goreleaser, provenance attestation, summary |

The end-to-end checks live in `scripts/integration-test.sh` rather than in
workflow YAML, so GitHub Actions and the GitLab pipeline cannot drift. The
script covers the roundtrip, tamper detection, rejection under an unrelated
key, the "`exp` is advisory" contract, and schema validation failures.
