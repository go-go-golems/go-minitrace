package serve

import (
	"slices"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func TestDetectBadgesAndArtifacts(t *testing.T) {
	commitCommand := `git commit -m "feat: wire serve frontend"`
	ticketCommand := `docmgr ticket create --ticket WESEN-OS-002`
	docCommand := `docmgr doc add --title "Serve diary"`
	diaryPath := "/tmp/project/ttmp/2026/04/01/reference/01-diary.md"
	diaryCommand := "apply_patch diary"
	errorCommand := "exec false"

	durationMS := 5
	commitCall := minitrace.BuildToolCall(
		"commit-1",
		intPtr(1),
		nil,
		"exec_command",
		"execute",
		nil,
		nil,
		map[string]any{"cmd": commitCommand},
		true,
		"",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)
	duplicateCommitCall := minitrace.BuildToolCall(
		"commit-2",
		intPtr(1),
		nil,
		"exec_command",
		"execute",
		nil,
		nil,
		map[string]any{"cmd": commitCommand},
		true,
		"",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)
	ticketCall := minitrace.BuildToolCall(
		"ticket-1",
		intPtr(1),
		nil,
		"exec_command",
		"execute",
		nil,
		nil,
		map[string]any{"cmd": ticketCommand},
		true,
		"",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)
	docCall := minitrace.BuildToolCall(
		"doc-1",
		intPtr(1),
		nil,
		"exec_command",
		"execute",
		nil,
		nil,
		map[string]any{"cmd": docCommand},
		true,
		"",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)
	diaryCall := minitrace.BuildToolCall(
		"diary-1",
		intPtr(1),
		nil,
		"apply_patch",
		"modify",
		&diaryPath,
		nil,
		map[string]any{"cmd": diaryCommand},
		true,
		"",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)
	errorCall := minitrace.BuildToolCall(
		"error-1",
		intPtr(1),
		nil,
		"exec_command",
		"execute",
		nil,
		nil,
		map[string]any{"cmd": errorCommand},
		false,
		"",
		nil,
		&durationMS,
		nil,
		nil,
		nil,
		nil,
	)

	if badges := DetectBadges(commitCall); !slices.Contains(badges, BadgeCommit) {
		t.Fatalf("expected commit badge, got %v", badges)
	}
	if badges := DetectBadges(ticketCall); !slices.Contains(badges, BadgeTicketCreate) {
		t.Fatalf("expected ticket-create badge, got %v", badges)
	}
	if badges := DetectBadges(docCall); !slices.Contains(badges, BadgeDocAdd) {
		t.Fatalf("expected doc-add badge, got %v", badges)
	}
	if badges := DetectBadges(diaryCall); !slices.Contains(badges, BadgeDiaryWrite) {
		t.Fatalf("expected diary-write badge, got %v", badges)
	}
	if badges := DetectBadges(errorCall); !slices.Contains(badges, BadgeError) {
		t.Fatalf("expected error badge, got %v", badges)
	}

	tcByID := map[string]minitrace.ToolCall{
		commitCall.ID:          commitCall,
		duplicateCommitCall.ID: duplicateCommitCall,
		ticketCall.ID:          ticketCall,
		docCall.ID:             docCall,
		diaryCall.ID:           diaryCall,
	}
	block := rawSessionBlock{
		Turns: []rawBlockTurn{
			{ToolCallIDs: []string{commitCall.ID, duplicateCommitCall.ID, ticketCall.ID}},
			{ToolCallIDs: []string{docCall.ID, diaryCall.ID}},
		},
	}

	artifacts := DetectBlockArtifacts(block, tcByID)
	if len(artifacts.Commits) != 1 || artifacts.Commits[0] != "feat: wire serve frontend" {
		t.Fatalf("expected one deduplicated commit artifact, got %+v", artifacts.Commits)
	}
	if len(artifacts.TicketsCreated) != 1 || artifacts.TicketsCreated[0] != "WESEN-OS-002" {
		t.Fatalf("expected ticket artifact, got %+v", artifacts.TicketsCreated)
	}
	if len(artifacts.DocsAdded) != 1 || artifacts.DocsAdded[0] != "Serve diary" {
		t.Fatalf("expected doc artifact, got %+v", artifacts.DocsAdded)
	}
	if artifacts.DiaryWrites != 1 {
		t.Fatalf("expected diary write count 1, got %d", artifacts.DiaryWrites)
	}
}
