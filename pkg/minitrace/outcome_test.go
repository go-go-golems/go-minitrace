package minitrace

import (
	"encoding/json"
	"testing"
)

func TestToolOutcomeNullability(t *testing.T) {
	for _, status := range []ToolOutcomeStatus{ToolOutcomeUnknown, ToolOutcomePending, ToolOutcomeCancelled} {
		t.Run(string(status), func(t *testing.T) {
			output := ToolCallOutput{Status: status}
			if output.Succeeded() || output.Failed() || output.OutcomeStatus() != status {
				t.Fatalf("non-binary outcome became binary: %#v", output)
			}
			data, err := json.Marshal(output)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded["success"] != nil || decoded["status"] != string(status) {
				t.Fatalf("wire outcome is not nullable: %s", data)
			}
		})
	}
	for _, success := range []bool{true, false} {
		var output ToolCallOutput
		output.SetSuccess(success)
		if output.Success == nil || *output.Success != success || output.Succeeded() != success || output.Failed() == success {
			t.Fatalf("known outcome changed: %#v", output)
		}
		if output.OutcomeStatus() != output.Status {
			t.Fatalf("status disagrees with success: %#v", output)
		}
	}
}

func TestToolOutcomeAbsentEvidenceIsUnknown(t *testing.T) {
	for _, data := range []string{`{}`, `{"success":null}`, `{"status":"succeeded"}`, `{"status":"unrecognized"}`} {
		var output ToolCallOutput
		if err := json.Unmarshal([]byte(data), &output); err != nil {
			t.Fatal(err)
		}
		if output.OutcomeStatus() != ToolOutcomeUnknown || output.Succeeded() || output.Failed() {
			t.Fatalf("missing binary evidence became known: %s", data)
		}
	}
}
