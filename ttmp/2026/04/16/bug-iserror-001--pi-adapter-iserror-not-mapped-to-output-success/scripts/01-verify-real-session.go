package main

import (
	"fmt"
	"os"
	"strings"

	piadapter "github.com/go-go-golems/go-minitrace/pkg/adapters/pi"
)

const defaultSessionPath = "/home/manuel/.pi/agent/sessions/--home-manuel-code-wesen-crib-k3s--/2026-04-16T01-34-34-242Z_2035dd97-cfb1-47ba-a90d-41096ae624d5.jsonl"

func main() {
	sourcePath := defaultSessionPath
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		sourcePath = os.Args[1]
	}

	session, err := piadapter.ConvertFile(sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "convert failed: %v\n", err)
		os.Exit(1)
	}

	failures := 0
	fileNotFound := 0
	for _, tc := range session.ToolCalls {
		if !tc.Output.Failed() {
			continue
		}
		failures++
		if tc.Output.Error != nil && strings.Contains(*tc.Output.Error, "File not found") {
			fileNotFound++
		}
	}

	fmt.Printf("session_id: %s\n", session.ID)
	fmt.Printf("tool_calls: %d\n", len(session.ToolCalls))
	fmt.Printf("failed_tool_calls: %d\n", failures)
	fmt.Printf("file_not_found_failures: %d\n", fileNotFound)
	fmt.Println()
	fmt.Println("sample_failures:")

	shown := 0
	for _, tc := range session.ToolCalls {
		if !tc.Output.Failed() {
			continue
		}
		errorText := ""
		if tc.Output.Error != nil {
			errorText = *tc.Output.Error
		}
		fmt.Printf("- id=%s tool=%s error=%q\n", tc.ID, tc.ToolName, errorText)
		shown++
		if shown >= 20 {
			break
		}
	}
}
