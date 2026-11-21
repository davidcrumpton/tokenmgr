module github.com/davidcrumpton/tokenmgr

go 1.24.0

// v0.1.1's module zip is unusable: it contains
// examples/:NodeRed:ValideToken.json, and Go rejects module paths containing
// ':', so `go install ...@v0.1.1` fails with "malformed file path". The tag
// and the release artifacts are fine; only the Go module is affected. Renamed
// in v0.1.2.
retract v0.1.1

require (
	github.com/goccy/go-yaml v1.19.2
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
