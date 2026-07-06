package discover

import (
	"testing"
	"time"

	"github.com/go-go-golems/go-minitrace/pkg/adapters"
)

func TestParseSinceAcceptsRFC3339AndDate(t *testing.T) {
	since, err := parseSince("2026-03-29T11:00:00Z")
	if err != nil || since == nil {
		t.Fatalf("expected RFC3339 to parse, got %v (%v)", since, err)
	}
	if !since.Equal(time.Date(2026, 3, 29, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected RFC3339 parse result: %v", since)
	}

	since, err = parseSince("2026-03-29")
	if err != nil || since == nil {
		t.Fatalf("expected YYYY-MM-DD to parse, got %v (%v)", since, err)
	}
	if !since.Equal(time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected date parse result: %v", since)
	}

	since, err = parseSince("")
	if err != nil || since != nil {
		t.Fatalf("expected empty --since to yield nil, got %v (%v)", since, err)
	}

	if _, err := parseSince("yesterday"); err == nil {
		t.Fatalf("expected invalid --since value to error")
	}
}

func TestKeepLocatorCwdContains(t *testing.T) {
	locator := adapters.SessionLocator{Cwd: "/home/manuel/workspaces/project"}
	if !keepLocator(locator, "workspaces", nil) {
		t.Fatalf("expected matching substring to keep locator")
	}
	if keepLocator(locator, "Workspaces", nil) {
		t.Fatalf("expected case-sensitive mismatch to drop locator")
	}
	if keepLocator(adapters.SessionLocator{Cwd: ""}, "workspaces", nil) {
		t.Fatalf("expected empty cwd to drop locator when filter is set")
	}
	if !keepLocator(adapters.SessionLocator{Cwd: ""}, "", nil) {
		t.Fatalf("expected no filter to keep locator")
	}
}

func TestKeepLocatorSince(t *testing.T) {
	since, err := parseSince("2026-04-01")
	if err != nil {
		t.Fatalf("parseSince returned error: %v", err)
	}

	newer := adapters.SessionLocator{StartedAt: "2026-04-02T10:00:00Z"}
	older := adapters.SessionLocator{StartedAt: "2026-03-31T23:59:59Z"}
	boundary := adapters.SessionLocator{StartedAt: "2026-04-01T00:00:00Z"}
	empty := adapters.SessionLocator{}

	if !keepLocator(newer, "", since) {
		t.Fatalf("expected newer session to be kept")
	}
	if keepLocator(older, "", since) {
		t.Fatalf("expected older session to be dropped")
	}
	if !keepLocator(boundary, "", since) {
		t.Fatalf("expected session at boundary to be kept (>= semantics)")
	}
	if keepLocator(empty, "", since) {
		t.Fatalf("expected session without started_at to be dropped when --since is set")
	}
	if !keepLocator(empty, "", nil) {
		t.Fatalf("expected session without started_at to be kept when --since is unset")
	}
}
