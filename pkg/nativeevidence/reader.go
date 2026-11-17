// Package nativeevidence provides opt-in, provenance-preserving native JSONL
// inspection. It does not execute transcript contents or reconstruct model memory.
package nativeevidence

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

const DefaultMaxRecordBytes = 16 << 20
const DefaultMaxIndexBytes = 64 << 20

// Options limits raw record allocation and logical retained JSON/index bytes.
// MaxIndexBytes is not a promise about exact Go heap usage.
type Options struct {
	MaxRecordBytes int   `json:"maxRecordBytes"`
	MaxIndexBytes  int64 `json:"maxIndexBytes"`
}

func (o Options) normalized() (Options, error) {
	if o.MaxRecordBytes < 0 || o.MaxIndexBytes < 0 {
		return o, errors.New("native limits must be nonnegative")
	}
	if o.MaxRecordBytes == 0 {
		o.MaxRecordBytes = DefaultMaxRecordBytes
	}
	if o.MaxIndexBytes == 0 {
		o.MaxIndexBytes = DefaultMaxIndexBytes
	}
	return o, nil
}

// Coordinate identifies physical source bytes, not conversation chronology.
type Coordinate struct {
	Line       int64 `json:"line"`
	ByteOffset int64 `json:"byteOffset"`
	RawBytes   int64 `json:"rawBytes"`
}
type Diagnostic struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Reference string `json:"reference,omitempty"`
}
type Record struct {
	Source     Coordinate     `json:"source"`
	Value      map[string]any `json:"value,omitempty"`
	RawJSON    string         `json:"rawJSON,omitempty"`
	Diagnostic *Diagnostic    `json:"diagnostic,omitempty"`
}
type ScanStats struct {
	SHA256     string `json:"sha256"`
	HashScope  string `json:"hashScope"`
	Bytes      int64  `json:"bytes"`
	Lines      int64  `json:"lines"`
	Malformed  int64  `json:"malformed"`
	StopReason string `json:"stopReason"`
}

// ScanFile always closes the file, including callback errors, cancellation and
// early stop. Returning false from visit stops at the consumed record boundary.
func ScanFile(ctx context.Context, path string, opts Options, visit func(Record) (bool, error)) (ScanStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return ScanStats{}, err
	}
	defer func() { _ = f.Close() }()
	return Scan(ctx, f, opts, visit)
}

// Scan hashes exact consumed bytes, including malformed records and line endings.
// Only EOF establishes hashScope=whole_source; an early hash is a consumed prefix.
func Scan(ctx context.Context, input io.Reader, opts Options, visit func(Record) (bool, error)) (stats ScanStats, retErr error) {
	opts, retErr = opts.normalized()
	if retErr != nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	h := sha256.New()
	stats.HashScope = "consumed_prefix"
	stats.StopReason = "error"
	defer func() { stats.SHA256 = hex.EncodeToString(h.Sum(nil)) }()
	reader := bufio.NewReaderSize(input, 32<<10)
	for {
		start := stats.Bytes
		var raw []byte
		oversized := false
		eof := false
		for {
			if err := ctx.Err(); err != nil {
				stats.StopReason = "canceled"
				return stats, err
			}
			part, err := reader.ReadSlice('\n')
			_, _ = h.Write(part)
			stats.Bytes += int64(len(part))
			if !oversized {
				if int64(len(raw))+int64(len(part)) > int64(opts.MaxRecordBytes) {
					oversized = true
					raw = nil
				} else {
					raw = append(raw, part...)
				}
			}
			if errors.Is(err, bufio.ErrBufferFull) {
				continue
			}
			if errors.Is(err, io.EOF) {
				eof = true
			} else if err != nil {
				return stats, err
			}
			break
		}
		if stats.Bytes == start && eof {
			stats.HashScope = "whole_source"
			stats.StopReason = "eof"
			return stats, nil
		}
		stats.Lines++
		record := Record{Source: Coordinate{stats.Lines, start, stats.Bytes - start}}
		switch {
		case oversized:
			record.Diagnostic = &Diagnostic{Code: "record_too_large", Message: fmt.Sprintf("record exceeds %d raw bytes", opts.MaxRecordBytes)}
		case !utf8.Valid(raw):
			record.Diagnostic = &Diagnostic{Code: "invalid_utf8", Message: "record contains invalid UTF-8; original bytes remain included in hash"}
		default:
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			err := decoder.Decode(&record.Value)
			var trailing any
			trailingErr := decoder.Decode(&trailing)
			if err != nil || record.Value == nil || trailingErr != io.EOF {
				record.Value = nil
				record.Diagnostic = &Diagnostic{Code: "invalid_record", Message: "expected a JSON object"}
			} else {
				record.RawJSON = string(raw)
			}
		}
		if record.Diagnostic != nil {
			stats.Malformed++
		}
		if eof {
			stats.HashScope = "whole_source"
		}
		if visit != nil {
			more, err := visit(record)
			if err != nil {
				return stats, err
			}
			if !more {
				stats.StopReason = "callback"
				return stats, nil
			}
		}
		if eof {
			stats.StopReason = "eof"
			return stats, nil
		}
	}
}
