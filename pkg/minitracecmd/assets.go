package minitracecmd

import "embed"

//go:embed core
var embeddedCommandsFS embed.FS

func EmbeddedSourceRoot() SourceRoot {
	return SourceRoot{
		Name:     "embedded",
		FS:       embeddedCommandsFS,
		RootDir:  "core",
		Readonly: true,
	}
}

func LoadEmbeddedCatalog() (*Catalog, error) {
	return LoadCatalog([]SourceRoot{EmbeddedSourceRoot()})
}
