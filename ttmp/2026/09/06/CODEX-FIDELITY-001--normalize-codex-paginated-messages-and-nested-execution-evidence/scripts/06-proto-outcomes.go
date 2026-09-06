// Emit a synthetic protojson response for the independent TypeScript decoder check.
package main

import (
	"fmt"
	"os"

	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	var calls []*apiv1.ToolCall
	for _, status := range []minitrace.ToolOutcomeStatus{
		minitrace.ToolOutcomeUnknown, minitrace.ToolOutcomePending, minitrace.ToolOutcomeCancelled,
		minitrace.ToolOutcomeSucceeded, minitrace.ToolOutcomeFailed,
	} {
		output := minitrace.ToolCallOutput{Status: status}
		if status == minitrace.ToolOutcomeSucceeded || status == minitrace.ToolOutcomeFailed {
			output.SetSuccess(status == minitrace.ToolOutcomeSucceeded)
		}
		wire := &apiv1.ToolCallOutput{Success: output.Success, Status: string(output.OutcomeStatus())}
		if output.Failed() {
			exitCode := int32(-1)
			wire.ExitCode = &exitCode
		}
		calls = append(calls, &apiv1.ToolCall{Id: string(status), ToolName: "exec_command", Output: wire})
	}
	response := &apiv1.GetSessionBlocksResponse{Blocks: []*apiv1.SessionBlock{{
		Turns: []*apiv1.Turn{{Role: "assistant", ToolCallsInTurn: calls}},
	}}}
	data, err := protojson.Marshal(response)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
