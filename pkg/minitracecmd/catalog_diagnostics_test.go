package minitracecmd

import (
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"testing"
	"testing/fstest"
)

func TestCatalogIsolatesMalformedSiblingAtDiscoveryAndRuntime(t *testing.T) {
	roots := []SourceRoot{{Name: "test", RootDir: ".", FS: fstest.MapFS{
		"good.js": {Data: []byte(`function good(){return {ok:true}}; __verb__("good",{name:"good",short:"Good"});`)},
		"bad.js":  {Data: []byte(`function bad(){}; __verb__("bad",{name:"bad",short:"Bad",fields:{limit:{type:"int",default:-1}}});`)},
	}}}
	if _, err := LoadCatalog(roots); err == nil {
		t.Fatal("strict catalog should reject unsupported signed metadata")
	}
	catalog, err := LoadCatalogWithDiagnostics(roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Diagnostics) != 1 || catalog.ByName["good"] == nil || catalog.ByName["bad"] != nil {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	registry, err := jsverbs.ScanFS(catalog.RuntimeSourceFS("test"), ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Verbs()) != 1 {
		t.Fatalf("runtime verbs: %d", len(registry.Verbs()))
	}
}
