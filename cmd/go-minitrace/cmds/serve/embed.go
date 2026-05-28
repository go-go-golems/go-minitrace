package serve

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend
var frontendEmbedFS embed.FS

var frontendFS = mustSubFS(frontendEmbedFS, "frontend")

func init() {
	// Detect missing frontend build: when installed via `go install` without
	// running `go generate` first, the embedded frontend only contains .gitkeep.
	if _, err := frontendFS.Open("index.html"); err != nil {
		log.Error().Msg(
			"Embedded frontend is missing index.html. " +
				"The web UI will not work. " +
				"Run `go generate ./cmd/go-minitrace/cmds/serve` before building, " +
				"or download a release binary from GitHub.")
	}
}

func mustSubFS(fsys fs.FS, dir string) fs.FS {
	subFS, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return subFS
}
