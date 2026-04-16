package serve

//go:generate go run ../../../build-web

// This package-level generator refreshes the embedded frontend bundle used by
// go-minitrace serve. Run `go generate ./...` (or `go generate
// ./cmd/go-minitrace/cmds/serve`) before release/build workflows that need a
// fresh production UI bundle.
