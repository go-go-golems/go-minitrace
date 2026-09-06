package serve

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestToolOutcomeProtoPresenceAndBadges(t *testing.T) {
	for _, status := range []minitrace.ToolOutcomeStatus{
		minitrace.ToolOutcomeUnknown, minitrace.ToolOutcomePending, minitrace.ToolOutcomeCancelled,
		minitrace.ToolOutcomeSucceeded, minitrace.ToolOutcomeFailed,
	} {
		t.Run(string(status), func(t *testing.T) {
			output := minitrace.ToolCallOutput{Status: status}
			if status == minitrace.ToolOutcomeSucceeded || status == minitrace.ToolOutcomeFailed {
				output.SetSuccess(status == minitrace.ToolOutcomeSucceeded)
			}
			apiOutput := protoToolCallOutput(ToolCallOutput{Success: output.Success, Status: string(output.OutcomeStatus())})
			data, err := protojson.Marshal(apiOutput)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			if wire["status"] != string(status) {
				t.Fatalf("status lost in protojson: %s", data)
			}
			if output.Success == nil {
				if _, exists := wire["success"]; exists {
					t.Fatalf("unknown outcome must not encode a binary value: %s", data)
				}
			} else if wire["success"] != *output.Success {
				t.Fatalf("known binary outcome lost: %s", data)
			}
			badges := DetectBadges(minitrace.ToolCall{Output: output})
			if slices.Contains(badges, BadgeError) != (status == minitrace.ToolOutcomeFailed) {
				t.Fatalf("unknown outcome became error badge: %v", badges)
			}
		})
	}
}

func TestToolOutcomeProtoPreservesSignedExitCode(t *testing.T) {
	code := -1
	output := protoToolCallOutput(ToolCallOutput{ExitCode: &code})
	if output.ExitCode == nil || *output.ExitCode != -1 {
		t.Fatalf("signed exit code lost: %+v", output)
	}
}
