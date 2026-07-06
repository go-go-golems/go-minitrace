package minitrace

import (
	"crypto/sha256"
	"encoding/hex"
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

func TestTruncateContentReportsFullSizeAndHashForLargeInput(t *testing.T) {
	// Larger than TruncateLimit*4 (40 KiB): the previous implementation
	// pre-capped the input before computing full_bytes and the hash.
	input := strings.Repeat("a", TruncateLimit*5)

	truncated, fullBytes, fullHash := TruncateContent(input, TruncateLimit)
	if truncated == nil {
		t.Fatalf("expected truncated content")
	}
	if fullBytes == nil || *fullBytes != len(input) {
		t.Fatalf("expected full_bytes %d, got %v", len(input), fullBytes)
	}
	sum := sha256.Sum256([]byte(input))
	expectedHash := "sha256:" + hex.EncodeToString(sum[:])
	if fullHash == nil || *fullHash != expectedHash {
		t.Fatalf("expected hash of full content %s, got %v", expectedHash, fullHash)
	}
	if len(*truncated) > TruncateLimit+len("\n[truncated]") {
		t.Fatalf("expected stored content capped at limit, got %d bytes", len(*truncated))
	}
}

func TestDurationBetweenMS(t *testing.T) {
	start := "2026-03-29T10:00:00Z"
	end := "2026-03-29T10:00:02.500Z"

	durationMS := DurationBetweenMS(&start, &end)
	if durationMS == nil || *durationMS != 2500 {
		t.Fatalf("expected 2500ms, got %v", durationMS)
	}
	if DurationBetweenMS(&end, &start) != nil {
		t.Fatalf("expected nil for end before start")
	}
	if DurationBetweenMS(nil, &end) != nil || DurationBetweenMS(&start, nil) != nil {
		t.Fatalf("expected nil for missing timestamps")
	}
	invalid := "not-a-timestamp"
	if DurationBetweenMS(&invalid, &end) != nil {
		t.Fatalf("expected nil for unparseable start timestamp")
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
	if session.Events == nil {
		t.Fatalf("expected non-nil events slice")
	}
	if session.Attachments == nil {
		t.Fatalf("expected non-nil attachments slice")
	}
}

func TestBuildEventDefaults(t *testing.T) {
	timestamp := "2026-06-10T12:00:00Z"
	raw := map[string]any{"type": "compaction"}
	event := BuildEvent("event-1", &timestamp, "compaction", "Compaction", "Context compacted", raw)

	if event.ID != "event-1" || event.Kind != "compaction" {
		t.Fatalf("unexpected event identity: %+v", event)
	}
	if event.Timestamp == nil || *event.Timestamp != timestamp {
		t.Fatalf("unexpected event timestamp: %+v", event.Timestamp)
	}
	if event.Severity != "info" {
		t.Fatalf("expected info severity, got %q", event.Severity)
	}
	if !event.CollapsedByDefault {
		t.Fatalf("expected source events to collapse by default")
	}
	if event.RawJSON == nil {
		t.Fatalf("expected raw source payload")
	}
}

func TestBuildAttachmentDefaults(t *testing.T) {
	timestamp := "2026-06-10T12:00:00Z"
	raw := map[string]any{"type": "attachment"}
	attachment := BuildAttachment("attachment-1", &timestamp, "image", "screenshot.png", "image/png", raw)

	if attachment.ID != "attachment-1" || attachment.Kind != "image" {
		t.Fatalf("unexpected attachment identity: %+v", attachment)
	}
	if attachment.Timestamp == nil || *attachment.Timestamp != timestamp {
		t.Fatalf("unexpected attachment timestamp: %+v", attachment.Timestamp)
	}
	if attachment.Name != "screenshot.png" || attachment.MediaType != "image/png" {
		t.Fatalf("unexpected attachment labels: %+v", attachment)
	}
	if attachment.RawJSON == nil {
		t.Fatalf("expected raw source payload")
	}
}
