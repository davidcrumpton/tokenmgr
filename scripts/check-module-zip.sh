#!/bin/sh
# Verify that the committed tree would produce a valid Go module zip.
#
# Why this exists: v0.1.1 shipped with examples/:NodeRed:ValideToken.json. Go
# rejects ':' in module file paths, so the module zip for that tag is unusable
# and `go install github.com/davidcrumpton/tokenmgr/cmd/tokenmgr@v0.1.1` fails
# with:
#
#   create zip: examples/:NodeRed:ValideToken.json: malformed file path
#   "examples/:NodeRed:ValideToken.json": invalid char ':'
#
# The tag, the tarballs and the packages were all fine — only the Go module was
# broken. `go build` and `go test` work on a checkout no matter what the files
# are called, which is exactly why this needs its own check.
#
# This uses golang.org/x/mod/zip, the same validator the module proxy runs,
# instead of reimplementing the character rules. The checker is built in a
# throwaway module so tokenmgr's own go.mod stays free of the dependency, and
# it validates a `git archive` export so untracked files (dist/) cannot
# influence the result.
#
# It also guards the other direction: the README logo must exist in the repo but
# must not appear in the export (see .gitattributes), because the module zip is
# built from that export and every `go install` would otherwise download it.
#
# Usage: scripts/check-module-zip.sh   (from anywhere)
set -eu

cd "$(dirname "$0")/.."

repo=$(pwd)
module_path=$(sed -n 's/^module \(.*\)$/\1/p' go.mod)
[ -n "$module_path" ] || { echo "could not read module path from go.mod" >&2; exit 1; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT INT TERM

# Export exactly what a tag would contain: tracked files only.
mkdir -p "$work/tree"
git archive --format=tar HEAD | tar -x -C "$work/tree"

# The README logo is a repo-only asset and must stay out of the module, or every
# `go install` downloads it. That is enforced by `images/ export-ignore` in
# .gitattributes, which git archive honours (and so does the module proxy, which
# builds the zip from the same tree). Assert it here: without this, deleting
# that one line silently ships the image to every user and nothing fails.
if [ -e "$work/tree/images" ]; then
    echo "images/ is in the tag export, so it would ship in the module zip:" >&2
    ls -l "$work/tree/images" >&2
    echo "check the 'images/ export-ignore' line in .gitattributes" >&2
    exit 1
fi

# The asset is excluded from the export, so nothing downstream would notice it
# going missing while README.md still points at it.
if grep -q 'src="images/' README.md; then
    [ -f images/tokenmanager-logo.png ] || {
        echo "README.md references images/tokenmanager-logo.png but it is missing" >&2
        exit 1
    }
fi

cd "$work"
go mod init zipcheck >/dev/null 2>&1
# Pinned so a new x/mod release cannot change the verdict on its own.
if ! go get golang.org/x/mod@v0.31.0 >/dev/null 2>&1; then
    echo "could not fetch golang.org/x/mod (network?)" >&2
    exit 1
fi

cat >main.go <<'GO'
package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

func main() {
	dir, modulePath, version := os.Args[1], os.Args[2], os.Args[3]
	// The zip itself is discarded: only the validation matters, and
	// CreateFromDir applies the same restrictions the module proxy does.
	if err := zip.CreateFromDir(io.Discard, module.Version{Path: modulePath, Version: version}, dir); err != nil {
		fmt.Fprintf(os.Stderr, "module zip would be rejected:\n%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("module zip for %s@%s is valid\n", modulePath, version)
}
GO

go mod tidy >/dev/null 2>&1
go run . "$work/tree" "$module_path" "v0.0.0"
