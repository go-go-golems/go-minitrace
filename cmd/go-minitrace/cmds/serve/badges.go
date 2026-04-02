package serve

import (
	"regexp"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

const (
	BadgeCommit       BadgeType = "commit"
	BadgeTicketCreate BadgeType = "ticket-create"
	BadgeDocAdd       BadgeType = "doc-add"
	BadgeDiaryWrite   BadgeType = "diary-write"
	BadgeError        BadgeType = "error"
)

var (
	commitMessagePattern = regexp.MustCompile(`git commit(?:\s+-[^\s]+\s+)*\s+-m\s+["']([^"']+)["']`)
	ticketPattern        = regexp.MustCompile(`--ticket\s+([A-Za-z0-9._-]+)`)
	titlePattern         = regexp.MustCompile(`--title\s+["']([^"']+)["']`)
)

func DetectBadges(toolCall minitrace.ToolCall) []BadgeType {
	badges := make([]BadgeType, 0, 2)
	command := strings.ToLower(extractCommand(toolCall))

	if !toolCall.Output.Success {
		badges = append(badges, BadgeError)
	}
	if strings.Contains(command, "git commit") {
		badges = append(badges, BadgeCommit)
	}
	if strings.Contains(command, "docmgr ticket create") || strings.Contains(command, "docmgr ticket create-ticket") {
		badges = append(badges, BadgeTicketCreate)
	}
	if strings.Contains(command, "docmgr doc add") {
		badges = append(badges, BadgeDocAdd)
	}
	if isDiaryWrite(toolCall, command) {
		badges = append(badges, BadgeDiaryWrite)
	}

	return badges
}

func DetectBlockArtifacts(block rawSessionBlock, tcByID map[string]minitrace.ToolCall) BlockArtifacts {
	artifacts := BlockArtifacts{
		Commits:        []string{},
		TicketsCreated: []string{},
		DocsAdded:      []string{},
		DiaryWrites:    0,
	}
	commitSeen := map[string]struct{}{}
	ticketSeen := map[string]struct{}{}
	docSeen := map[string]struct{}{}

	for _, turn := range block.Turns {
		for _, toolCallID := range turn.ToolCallIDs {
			toolCall, ok := tcByID[toolCallID]
			if !ok {
				continue
			}
			for _, badge := range DetectBadges(toolCall) {
				switch badge {
				case BadgeCommit:
					message := extractCommitMessage(toolCall)
					if message == "" {
						continue
					}
					if _, ok := commitSeen[message]; ok {
						continue
					}
					commitSeen[message] = struct{}{}
					artifacts.Commits = append(artifacts.Commits, message)
				case BadgeTicketCreate:
					ticketID := extractTicketID(toolCall)
					if ticketID == "" {
						continue
					}
					if _, ok := ticketSeen[ticketID]; ok {
						continue
					}
					ticketSeen[ticketID] = struct{}{}
					artifacts.TicketsCreated = append(artifacts.TicketsCreated, ticketID)
				case BadgeDocAdd:
					title := extractDocTitle(toolCall)
					if title == "" {
						continue
					}
					if _, ok := docSeen[title]; ok {
						continue
					}
					docSeen[title] = struct{}{}
					artifacts.DocsAdded = append(artifacts.DocsAdded, title)
				case BadgeDiaryWrite:
					artifacts.DiaryWrites++
				case BadgeError:
					continue
				}
			}
		}
	}

	return artifacts
}

func extractCommand(toolCall minitrace.ToolCall) string {
	if toolCall.Input.Command != nil {
		return *toolCall.Input.Command
	}
	if argMap, ok := toolCall.Input.Arguments.(map[string]any); ok {
		if command, ok := argMap["cmd"].(string); ok {
			return command
		}
	}
	return ""
}

func extractCommitMessage(toolCall minitrace.ToolCall) string {
	command := extractCommand(toolCall)
	matches := commitMessagePattern.FindStringSubmatch(command)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func extractTicketID(toolCall minitrace.ToolCall) string {
	command := extractCommand(toolCall)
	matches := ticketPattern.FindStringSubmatch(command)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func extractDocTitle(toolCall minitrace.ToolCall) string {
	command := extractCommand(toolCall)
	matches := titlePattern.FindStringSubmatch(command)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func isDiaryWrite(toolCall minitrace.ToolCall, command string) bool {
	filePath := ""
	if toolCall.Input.FilePath != nil {
		filePath = strings.ToLower(*toolCall.Input.FilePath)
	}

	if strings.Contains(filePath, "diary") && (toolCall.OperationType == "modify" || toolCall.OperationType == "create") {
		return true
	}
	if strings.Contains(command, "diary") || strings.Contains(command, "diary.md") {
		if strings.Contains(command, "apply_patch") || strings.Contains(command, "tee ") || strings.Contains(command, "cat >") || strings.Contains(command, "cat >>") {
			return true
		}
	}

	return false
}
