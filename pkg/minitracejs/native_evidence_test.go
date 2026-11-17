package minitracejs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestNativeEvidenceJS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"user\",\"uuid\":\"u\"}\n{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	_ = vm.Set("native", func() *goja.Object { return nativeEvidenceBuilderObject(context.Background(), vm) })
	_ = vm.Set("sourcePath", path)
	v, err := vm.RunString(`const seen=[]; const stats=native().File(sourcePath).ForEach(r=>{seen.push(r.source.line);return false}); JSON.stringify({seen,scope:stats.hashScope,reason:stats.stopReason})`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != `{"seen":[1],"scope":"consumed_prefix","reason":"callback"}` {
		t.Fatal(v.String())
	}
	v, err = vm.RunString(`native().File(sourcePath).Inspect().scan.hashScope`)
	if err != nil || v.String() != "whole_source" {
		t.Fatalf("%v %v", v, err)
	}
	for _, code := range []string{`native().Inspect()`, `native().MaxRecordBytes(1.5)`, `native().File(sourcePath).ForEach(1)`, `native().File(sourcePath).ForEach(()=>{throw new Error("stop-me")})`} {
		_, err = vm.RunString(code)
		if err == nil {
			t.Fatalf("expected error: %s", code)
		}
	}
	_, err = vm.RunString(`native().File(sourcePath).MaxRecordBytes(1).ForEach(r=>{if(!r.diagnostic)throw Error("missing diagnostic")})`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`native().File(sourcePath).ForEach(()=>{throw Error("visible failure")})`)
	if err == nil || !strings.Contains(err.Error(), "visible failure") {
		t.Fatal(err)
	}
}
