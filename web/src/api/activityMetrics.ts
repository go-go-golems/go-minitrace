import type { SessionMetrics } from "../gen/proto/go_go_golems/minitrace/api/v1/common_pb.js";

export function activityMetrics(metrics: SessionMetrics | undefined) {
  return {
    tool_call_record_count: metrics?.toolCallRecordCount ?? 0,
    orchestration_count: metrics?.orchestrationCount ?? 0,
    execution_record_count: metrics?.executionRecordCount ?? 0,
    file_change_count: metrics?.fileChangeCount ?? 0,
    model_invocation_count: metrics?.modelInvocationCount ?? 0,
    file_touch_count: metrics?.fileTouchCount ?? 0,
    confirmed_file_target_count: metrics?.confirmedFileTargetCount ?? 0,
  };
}
