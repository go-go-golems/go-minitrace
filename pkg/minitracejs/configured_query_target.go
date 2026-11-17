package minitracejs

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitracedb"
)

// NewArchiveQueryTargetWithCache makes the storage side effects of analysis explicit.
// Memory and none never create a disk cache; auto preserves the existing default.
func NewArchiveQueryTargetWithCache(ctx context.Context, archiveGlobs []string, query minitracedb.QueryOptions, mode, dir string) (*DBHandle, error) {
	if mode == "" {
		mode = "auto"
	}
	switch mode {
	case "auto", "memory", "disk", "none":
	default:
		return nil, fmt.Errorf("invalid cache mode %q: want auto, memory, disk, or none", mode)
	}
	if dir != "" && (mode == "memory" || mode == "none") {
		return nil, fmt.Errorf("cache-dir requires auto or disk cache mode")
	}
	b := NewDBBuilder(ctx)
	b.autoConvert = true
	b.strictConversion = true
	b.applyCachePolicy(&CachePolicy{Mode: mode, Dir: dir})
	b.query = minitracedb.WithDefaultQueryOptions(query)
	for _, pattern := range archiveGlobs {
		b.addGlob(pattern)
	}
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("minitrace.db: %v", b.errors)
	}
	if len(b.sources) == 0 {
		return nil, fmt.Errorf("archive globs matched no files: %s", strings.Join(archiveGlobs, ", "))
	}
	return b.Build()
}
