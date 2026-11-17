package nativeevidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// Blob preserves a provider object without treating its contents as instructions.
type Blob struct {
	Source  Coordinate     `json:"source"`
	Value   map[string]any `json:"value"`
	RawJSON string         `json:"rawJSON"`
}
type Request struct {
	ID          string       `json:"id"`
	QuerySource string       `json:"querySource"`
	Source      Coordinate   `json:"source"`
	ShapeHash   string       `json:"shapeHash"`
	Refs        []string     `json:"refs"`
	Status      string       `json:"status"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
type MessageCoordinate struct {
	Source     Coordinate `json:"source"`
	UUID       string     `json:"uuid"`
	ParentUUID string     `json:"parentUuid,omitempty"`
	Type       string     `json:"type"`
}
type Compaction struct {
	Boundary                MessageCoordinate   `json:"boundary"`
	LogicalParentUUID       string              `json:"logicalParentUuid,omitempty"`
	Metadata                map[string]any      `json:"metadata,omitempty"`
	Summaries               []MessageCoordinate `json:"summaries"`
	BeforeMessage           *MessageCoordinate  `json:"beforeMessage,omitempty"`
	FirstFollowingMessage   *MessageCoordinate  `json:"firstFollowingMessage,omitempty"`
	FirstFollowingAssistant *MessageCoordinate  `json:"firstFollowingAssistant,omitempty"`
}
type Inspection struct {
	Scan        ScanStats       `json:"scan"`
	Requests    []Request       `json:"requests"`
	Blobs       map[string]Blob `json:"blobs"`
	Shapes      map[string]Blob `json:"shapes"`
	Compactions []Compaction    `json:"compactions"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
	IndexBytes  int64           `json:"indexBytes"`
	Limitations []string        `json:"limitations"`
}

func str(v any) string { s, _ := v.(string); return s }
func message(r Record) MessageCoordinate {
	return MessageCoordinate{Source: r.Source, UUID: str(r.Value["uuid"]), ParentUUID: str(r.Value["parentUuid"]), Type: str(r.Value["type"])}
}

// InspectFile resolves requests using only records already observed in this file.
// Missing references are not filled from future or external files. Compaction
// metadata is exposed as evidence, never as a certified model replay boundary.
func InspectFile(ctx context.Context, path string, opts Options) (Inspection, error) {
	opts, err := opts.normalized()
	if err != nil {
		return Inspection{}, err
	}
	out := Inspection{Requests: []Request{}, Blobs: map[string]Blob{}, Shapes: map[string]Blob{}, Compactions: []Compaction{}, Diagnostics: []Diagnostic{}, Limitations: []string{"Request resolution is chronological and file-local; no future or cross-file reference recovery.", "Compaction coordinates are physical source positions, not exact causal replay; result-arrival provenance is not inferred.", "Index limit bounds logical retained JSON bytes, not exact heap usage. No instruction-wrapper extraction or model execution."}}
	charge := func(value any) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		n := int64(len(raw))
		if n > opts.MaxIndexBytes-out.IndexBytes {
			return fmt.Errorf("native index exceeds %d logical bytes", opts.MaxIndexBytes)
		}
		out.IndexBytes += n
		return nil
	}
	requests := map[string]int{}
	ambiguousBlobs := map[string]bool{}
	ambiguousShapes := map[string]bool{}
	var last *MessageCoordinate
	active := -1
	addDiag := func(d Diagnostic) error {
		if err := charge(d); err != nil {
			return err
		}
		out.Diagnostics = append(out.Diagnostics, d)
		return nil
	}
	out.Scan, err = ScanFile(ctx, path, opts, func(r Record) (bool, error) {
		if r.Diagnostic != nil {
			return true, addDiag(Diagnostic{Code: r.Diagnostic.Code, Message: fmt.Sprintf("line %d: %s", r.Source.Line, r.Diagnostic.Message)})
		}
		d := r.Value
		kind := str(d["type"])
		switch kind {
		case "api-request-blob", "api-request-shape":
			key := str(d["hash"])
			target := out.Blobs
			if kind == "api-request-shape" {
				key = str(d["shapeHash"])
				target = out.Shapes
			}
			if key == "" {
				return true, addDiag(Diagnostic{Code: "missing_identity", Message: fmt.Sprintf("line %d: %s lacks identity", r.Source.Line, kind)})
			}
			if prior, exists := target[key]; exists {
				payloadKey := "message"
				if kind == "api-request-shape" {
					payloadKey = "shape"
				}
				if reflect.DeepEqual(prior.Value[payloadKey], d[payloadKey]) {
					return true, nil
				}
				if kind == "api-request-shape" {
					ambiguousShapes[key] = true
				} else {
					ambiguousBlobs[key] = true
				}
				return true, addDiag(Diagnostic{Code: "duplicate_identity", Message: "first native object retained", Reference: key})
			}
			b := Blob{Source: r.Source, Value: d, RawJSON: r.RawJSON}
			if err := charge(b); err != nil {
				return false, err
			}
			target[key] = b
		case "api-request":
			req := Request{ID: str(d["id"]), QuerySource: str(d["querySource"]), Source: r.Source, ShapeHash: str(d["shapeHash"]), Refs: []string{}, Status: "complete", Diagnostics: []Diagnostic{}}
			diag := func(code, msg, ref string) {
				req.Diagnostics = append(req.Diagnostics, Diagnostic{code, msg, ref})
				if req.Status == "complete" {
					req.Status = code
				}
			}
			if req.ID == "" {
				diag("unsupported", "request lacks string id", "")
			}
			if _, ok := out.Shapes[req.ShapeHash]; !ok {
				diag("missing_shape", "request shape is not available", req.ShapeHash)
			}
			if ambiguousShapes[req.ShapeHash] {
				diag("unsupported", "duplicate shape identity is ambiguous", req.ShapeHash)
			}
			keepNumber, ok := d["keep"].(json.Number)
			keep, keepErr := keepNumber.Int64()
			if !ok || keepErr != nil || keep < 0 {
				diag("unsupported", "keep must be a nonnegative integer", "")
			} else if keep > 0 {
				idx, exists := requests[str(d["base"])]
				if !exists {
					diag("missing_base", "request base is not available", str(d["base"]))
				} else if keep > int64(len(out.Requests[idx].Refs)) {
					diag("missing_base", "base prefix is unavailable or shorter than keep", str(d["base"]))
				} else {
					// Bound copied reference bytes before allocating a potentially large slice.
					var n int64
					for _, ref := range out.Requests[idx].Refs[:int(keep)] {
						n += int64(len(ref) + 3)
					}
					if n > opts.MaxIndexBytes-out.IndexBytes {
						return false, errors.New("native index reference prefix exceeds limit")
					}
					req.Refs = append(req.Refs, out.Requests[idx].Refs[:int(keep)]...)
					for _, d := range out.Requests[idx].Diagnostics {
						if d.Code == "missing_base" || d.Code == "unsupported" {
							diag("missing_base", "base reference sequence is incomplete", str(r.Value["base"]))
							break
						}
					}
				}
			}
			tail, ok := d["tail"].([]any)
			if !ok {
				diag("unsupported", "tail must be an array", "")
			} else {
				for _, v := range tail {
					ref, valid := v.(string)
					if !valid {
						diag("unsupported", "tail contains a non-string reference", "")
						continue
					}
					req.Refs = append(req.Refs, ref)
				}
			}
			for _, ref := range req.Refs {
				if ambiguousBlobs[ref] {
					diag("unsupported", "duplicate blob identity is ambiguous", ref)
				}
				if _, ok := out.Blobs[ref]; !ok {
					diag("missing_blob", "referenced blob is not available", ref)
				}
			}
			if _, exists := requests[req.ID]; exists && req.ID != "" {
				diag("unsupported", "duplicate request id; first request remains the base lookup", req.ID)
			}
			if err := charge(req); err != nil {
				return false, err
			}
			out.Requests = append(out.Requests, req)
			if _, exists := requests[req.ID]; !exists && req.ID != "" {
				requests[req.ID] = len(out.Requests) - 1
			}
		}
		if kind == "system" && str(d["subtype"]) == "compact_boundary" {
			meta, _ := d["compactMetadata"].(map[string]any)
			c := Compaction{Boundary: message(r), LogicalParentUUID: str(d["logicalParentUuid"]), Metadata: meta, Summaries: []MessageCoordinate{}, BeforeMessage: last}
			if err := charge(c); err != nil {
				return false, err
			}
			out.Compactions = append(out.Compactions, c)
			active = len(out.Compactions) - 1
		}
		if kind == "user" || kind == "assistant" {
			m := message(r)
			if active >= 0 {
				c := &out.Compactions[active]
				if c.FirstFollowingMessage == nil {
					if err := charge(m); err != nil {
						return false, err
					}
					copy := m
					c.FirstFollowingMessage = &copy
				}
				if d["isCompactSummary"] == true {
					if err := charge(m); err != nil {
						return false, err
					}
					c.Summaries = append(c.Summaries, m)
				}
				if kind == "assistant" && c.FirstFollowingAssistant == nil {
					if err := charge(m); err != nil {
						return false, err
					}
					copy := m
					c.FirstFollowingAssistant = &copy
					active = -1
				}
			}
			last = &m
		}
		return true, nil
	})
	return out, err
}
