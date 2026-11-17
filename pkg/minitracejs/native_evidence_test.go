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

func TestNativeCompactionProjectionExample(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compaction.jsonl")
	records := []string{
		`{"type":"assistant","uuid":"old"}`,
		`{"type":"system","subtype":"compact_boundary","uuid":"boundary","logicalParentUuid":"attachment","compactMetadata":{"preservedSegment":{"headUuid":"old","tailUuid":"attachment"}}}`,
		`{"type":"user","uuid":"summary","parentUuid":"boundary","isCompactSummary":true}`,
		`{"type":"attachment","uuid":"attachment","parentUuid":"summary"}`,
		`{"type":"assistant","uuid":"new","parentUuid":"attachment"}`,
		`{"type":"system","subtype":"compact_boundary","uuid":"last"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(records, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	example, err := os.ReadFile("testdata/native_compaction_projection.js")
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	_ = vm.Set("native", func() *goja.Object { return nativeEvidenceBuilderObject(context.Background(), vm) })
	_ = vm.Set("sourcePath", path)
	if _, err = vm.RunString(`const inspection=native().File(sourcePath).Inspect();`); err != nil {
		t.Fatal(err)
	}
	projected, err := vm.RunString(string(example))
	if err != nil {
		t.Fatal(err)
	}
	_ = vm.Set("projected", projected)
	_, err = vm.RunString(`
 function check(v,m){if(!v)throw Error(m)}
 const c=projected[0];
 check(c.boundaryUuid==='boundary'&&c.boundary.line===2&&c.boundary.byteOffset>0,'boundary coordinates');
 check(c.logicalParentUuid==='attachment','logical parent');
 check(c.metadata.preservedSegment.headUuid==='old'&&c.metadata.preservedSegment.tailUuid==='attachment','preserved metadata');
 check(c.summaries.length===1&&c.summaries[0].uuid==='summary'&&c.summaries[0].source.line===3,'summary coordinates');
 check(c.precedingUuid==='old'&&c.firstFollowingUuid==='summary'&&c.firstFollowingAssistantUuid==='new','adjacent messages');
 const last=projected[1];
 check(last.logicalParentUuid===null&&last.metadata===null&&last.summaries.length===0,'optional metadata');
 check(last.firstFollowingUuid===null&&last.firstFollowingAssistantUuid===null,'EOF absence');
 `)
	if err != nil {
		t.Fatal(err)
	}
}
