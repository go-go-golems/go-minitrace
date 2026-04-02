package serve

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend
var frontendEmbedFS embed.FS

var frontendFS = mustSubFS(frontendEmbedFS, "frontend")

func mustSubFS(fsys fs.FS, dir string) fs.FS {
	subFS, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return subFS
}
