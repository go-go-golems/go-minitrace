package minitrace

import (
	"strings"
	"testing"
	"time"
)

func TestTruncateContentPreservesUTF8Boundary(t *testing.T) {
	input := strings.Repeat("🙂", 4096)

	truncated, fullBytes, fullHash := TruncateContent(input, 128)
	if truncated == nil {
		t.Fatalf("expected truncated content")
	}
	if fullBytes == nil || *fullBytes <= 128 {
		t.Fatalf("expected full byte count above limit, got %v", fullBytes)
	}
	if fullHash == nil || !strings.HasPrefix(*fullHash, "sha256:") {
		t.Fatalf("expected sha256 hash, got %v", fullHash)
	}
	if !strings.HasSuffix(*truncated, "\n[truncated]") {
		t.Fatalf("expected truncation marker, got %q", *truncated)
	}
}

func TestComputeTimingUsesMondayZeroConvention(t *testing.T) {
	start := time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)
	middle := start.Add(2 * time.Minute)
	afterIdle := start.Add(12 * time.Minute)

	timing := ComputeTiming([]time.Time{afterIdle, start, middle})

	if timing.DurationSeconds == nil || *timing.DurationSeconds != 720 {
		t.Fatalf("expected 720 second duration, got %+v", timing.DurationSeconds)
	}
	if timing.ActiveDurationSeconds == nil || *timing.ActiveDurationSeconds != 120 {
		t.Fatalf("expected 120 active seconds, got %+v", timing.ActiveDurationSeconds)
	}
	if timing.DayOfWeek == nil || *timing.DayOfWeek != 0 {
		t.Fatalf("expected Monday=0, got %+v", timing.DayOfWeek)
	}
}

func TestComputeMetricsPreservesNullSemanticsForGhostSessions(t *testing.T) {
	startedAt := "2026-03-28T12:00:00Z"
	endedAt := "2026-03-28T12:05:00Z"
	duration := 300.0
	timing := Timing{
		PrivacyLevel:          "full",
		DurationSeconds:       &duration,
		ActiveDurationSeconds: nil,
		StartedAt:             &startedAt,
		EndedAt:               &endedAt,
	}

	source := "human"
	turns := []Turn{
		BuildTurn(0, &startedAt, "user", &source, "hello"),
	}
	metrics := ComputeMetrics(turns, nil, timing, 0, nil)

	if metrics.ReadRatio != nil {
		t.Fatalf("expected nil read_ratio, got %v", *metrics.ReadRatio)
	}
	if metrics.TimeToFirstAction != nil {
		t.Fatalf("expected nil time_to_first_action, got %v", *metrics.TimeToFirstAction)
	}
	if metrics.ToolCallCount != 0 {
		t.Fatalf("expected zero tool calls, got %d", metrics.ToolCallCount)
	}
}

func TestBuildSessionSkeletonDefaults(t *testing.T) {
	session := BuildSessionSkeleton("session-1", "claude-code", "claude-code-jsonl-v2", "go-minitrace/dev")

	if session.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %s, got %s", SchemaVersion, session.SchemaVersion)
	}
	if session.Environment.AgentFramework == nil || *session.Environment.AgentFramework != "claude-code" {
		t.Fatalf("expected agent framework claude-code, got %+v", session.Environment.AgentFramework)
	}
	if session.Environment.ProviderHint == nil || *session.Environment.ProviderHint != "unknown" {
		t.Fatalf("expected provider hint unknown, got %+v", session.Environment.ProviderHint)
	}
	if session.Flags.NeedsCleaning != true {
		t.Fatalf("expected needs_cleaning=true")
	}
}
