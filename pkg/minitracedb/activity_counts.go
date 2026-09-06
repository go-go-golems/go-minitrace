package minitracedb

import (
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
	"strings"
)

var activityColumnNames = []string{"tool_call_record_count", "orchestration_count", "execution_record_count", "file_change_count", "model_invocation_count", "file_touch_count", "confirmed_file_target_count"}

func withActivityColumns(table TableDescriptor) TableDescriptor {
	var ddl strings.Builder
	for _, name := range activityColumnNames {
		table.Columns = append(table.Columns, ColumnDescriptor{Name: name, Type: "INTEGER", Nullable: true})
		ddl.WriteString("\t" + name + " INTEGER,\n")
	}
	table.CreateSQL = strings.Replace(table.CreateSQL, "(\n", "(\n"+ddl.String(), 1)
	return table
}

func activityValues(session *minitrace.Session) []any {
	// Derive these columns for older archives too; absent serialized counters
	// must not falsely report zero activity when tool records are available.
	c := minitrace.CountToolActivity(session.ToolCalls)
	return []any{c.ToolCallRecordCount, c.OrchestrationCount, c.ExecutionRecordCount, c.FileChangeCount, c.ModelInvocationCount, c.FileTouchCount, c.ConfirmedFileTargetCount}
}
