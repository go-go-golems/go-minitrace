package minitracejs

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

// corpusManifest is wired as mt.corpusManifest(paths) by the module.
func corpusManifest(vm *goja.Runtime, paths []string) map[string]any {
	manifest, err := minitracedb.BuildCorpusManifest(paths)
	if err != nil {
		panic(vm.NewGoError(err))
	}
	return toPlainMap(manifest)
}
