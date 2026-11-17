package nativeevidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, records ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "native.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(records, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestStreamingCoordinatesHashMalformedAndOversize(t *testing.T) {
	raw := "{\"type\":\"é\"}\r\n" + strings.Repeat("x", 70000) + "\n" + "{bad}\n" + "{\"x\":\"\xff\"}\n" + "{}"
	var rows []Record
	s, err := Scan(context.Background(), strings.NewReader(raw), Options{MaxRecordBytes: 32}, func(r Record) (bool, error) { rows = append(rows, r); return true, nil })
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(raw))
	if s.SHA256 != hex.EncodeToString(sum[:]) || s.HashScope != "whole_source" || s.Bytes != int64(len(raw)) || s.Malformed != 3 {
		t.Fatalf("stats %+v", s)
	}
	if len(rows) != 5 || rows[1].Source.ByteOffset != int64(len("{\"type\":\"é\"}\r\n")) || rows[1].Diagnostic.Code != "record_too_large" || rows[3].Diagnostic.Code != "invalid_utf8" || rows[4].Value == nil {
		t.Fatalf("records %+v", rows)
	}
}
func TestEarlyStopAndCancellation(t *testing.T) {
	s, err := Scan(context.Background(), strings.NewReader("{}\n{}\n"), Options{}, func(Record) (bool, error) { return false, nil })
	if err != nil || s.Bytes != 3 || s.HashScope != "consumed_prefix" || s.StopReason != "callback" {
		t.Fatalf("%+v %v", s, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s, err = Scan(ctx, strings.NewReader("{}\n"), Options{}, nil)
	if !errors.Is(err, context.Canceled) || s.Bytes != 0 || s.StopReason != "canceled" {
		t.Fatalf("%+v %v", s, err)
	}
	sentinel := errors.New("callback failure")
	_, err = Scan(context.Background(), strings.NewReader("{}\n"), Options{}, func(Record) (bool, error) { return false, sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
}
func TestRequestsAndCompactionGraph(t *testing.T) {
	p := fixture(t,
		`{"type":"api-request-shape","shapeHash":"s","shape":{"system":[]}}`,
		`{"type":"api-request-blob","hash":"a","message":{"content":"hello"}}`,
		`{"type":"api-request","id":"r1","querySource":"main","shapeHash":"s","keep":0,"tail":["a"]}`,
		`{"type":"api-request-blob","hash":"b","message":{"content":"world"}}`,
		`{"type":"api-request","id":"r2","shapeHash":"s","base":"r1","keep":1,"tail":["b"]}`,
		`{"type":"api-request","id":"r3","shapeHash":"s","base":"missing","keep":1,"tail":["lost"]}`,
		`{"type":"api-request","id":"r4","shapeHash":"no-shape","keep":0,"tail":[{}]}`,
		`{"type":"assistant","uuid":"old"}`,
		`{"type":"system","subtype":"compact_boundary","uuid":"boundary","logicalParentUuid":"later","compactMetadata":{"preservedSegment":{"headUuid":"old","tailUuid":"later"}}}`,
		`{"type":"user","uuid":"summary","parentUuid":"boundary","isCompactSummary":true}`,
		`{"type":"attachment","uuid":"later","parentUuid":"summary"}`,
		`{"type":"assistant","uuid":"new","parentUuid":"later"}`)
	out, err := InspectFile(context.Background(), p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Requests) != 4 || out.Requests[1].Status != "complete" || strings.Join(out.Requests[1].Refs, ",") != "a,b" {
		t.Fatalf("requests %+v", out.Requests)
	}
	if out.Requests[2].Status != "missing_base" || len(out.Requests[2].Diagnostics) != 2 || out.Requests[3].Status != "missing_shape" {
		t.Fatalf("diagnostics %+v", out.Requests)
	}
	c := out.Compactions[0]
	if c.BeforeMessage.UUID != "old" || c.Summaries[0].UUID != "summary" || c.FirstFollowingAssistant.UUID != "new" || c.LogicalParentUUID != "later" {
		t.Fatalf("compaction %+v", c)
	}
	if out.Blobs["a"].Value["message"].(map[string]any)["content"] != "hello" {
		t.Fatal("blob not preserved")
	}
	_, err = InspectFile(context.Background(), p, Options{MaxIndexBytes: 20})
	if err == nil || !strings.Contains(err.Error(), "index exceeds") {
		t.Fatalf("limit %v", err)
	}
}
func TestMissingBlobNotRecoveredFromFutureAndBadKeep(t *testing.T) {
	p := fixture(t, `{"type":"api-request-shape","shapeHash":"s"}`, `{"type":"api-request","id":"r","shapeHash":"s","keep":0,"tail":["future"]}`, `{"type":"api-request-blob","hash":"future","message":{}}`, `{"type":"api-request","id":"bad","shapeHash":"s","keep":0.5,"tail":[]}`)
	out, err := InspectFile(context.Background(), p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Requests[0].Status != "missing_blob" || out.Requests[1].Status != "unsupported" {
		t.Fatalf("%+v", out.Requests)
	}
	raw, err := json.Marshal(out)
	if err != nil || !strings.Contains(string(raw), `"hashScope":"whole_source"`) {
		t.Fatalf("%s %v", raw, err)
	}
}

func TestDuplicateIdentityAndHugeKeepAreNotComplete(t *testing.T) {
	p := fixture(t, `{"type":"api-request-shape","shapeHash":"s"}`, `{"type":"api-request-blob","hash":"a","message":{"content":"first"}}`, `{"type":"api-request-blob","hash":"a","message":{"content":"second"}}`, `{"type":"api-request","id":"r","shapeHash":"s","keep":0,"tail":["a"]}`, `{"type":"api-request","id":"huge","shapeHash":"s","keep":9223372036854775808,"tail":[]}`)
	out, err := InspectFile(context.Background(), p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Requests[0].Status != "unsupported" || out.Requests[1].Status != "unsupported" {
		t.Fatalf("%+v", out.Requests)
	}
	if out.Blobs["a"].Value["message"].(map[string]any)["content"] != "first" {
		t.Fatal("duplicate overwrote evidence")
	}
}

func TestExactJSONNumericEvidenceAndTrailingValues(t *testing.T) {
	raw := "{\"id\":9007199254740993}\r\n{} {}\n"
	var rows []Record
	_, err := Scan(context.Background(), strings.NewReader(raw), Options{}, func(r Record) (bool, error) { rows = append(rows, r); return true, nil })
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Value["id"].(json.Number).String() != "9007199254740993" || rows[0].RawJSON != "{\"id\":9007199254740993}\r\n" {
		t.Fatalf("numeric fidelity %+v", rows[0])
	}
	if rows[1].Diagnostic == nil || rows[1].Value != nil || rows[1].RawJSON != "" {
		t.Fatalf("trailing JSON accepted %+v", rows[1])
	}
}
