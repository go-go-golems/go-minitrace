package minitracejs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
)

func TestCorpusManifestJSContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.json")
	if err := os.WriteFile(path, []byte(`{"id":"manifest-test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	if err := vm.Set("manifest", func(paths []string) map[string]any { return corpusManifest(vm, paths) }); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("path", path); err != nil {
		t.Fatal(err)
	}
	value, err := vm.RunString(`const m=manifest([path,path]); m.entries.length===2 && m.duplicateCount===1 && m.entries[1].duplicateOf===0 && m.entries[0].sessionId==="manifest-test" && m.conflictCount===0`)
	if err != nil || !value.ToBoolean() {
		t.Fatalf("JS manifest contract: %v, %v", value, err)
	}
	value, err = vm.RunString(`let caught=false; try {manifest([path+".missing"])} catch(e) {caught=String(e).includes("manifest read")} caught`)
	if err != nil || !value.ToBoolean() {
		t.Fatalf("JS error contract: %v, %v", value, err)
	}
}
