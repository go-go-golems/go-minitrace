import { fromJson } from "@bufbuild/protobuf";
import { activityMetrics } from "./activityMetrics.ts";
import {
  ListSessionsResponseSchema,
  GetSessionSummaryResponseSchema,
  GetSessionBlocksResponseSchema,
  GetSessionDetailResponseSchema,
  type SessionSummary as PbSessionSummary,
  type SessionSummaryDetail as PbSessionSummaryDetail,
  type SessionBlock as PbSessionBlock,
  type SessionDetail as PbSessionDetail,
  type SessionEvent as PbSessionEvent,
  type SessionAttachment as PbSessionAttachment,
  type Turn as PbTurn,
  type ToolCall as PbToolCall,
} from "../gen/proto/go_go_golems/minitrace/api/v1/sessions_pb.js";
import { ToolCallBadge } from "../gen/proto/go_go_golems/minitrace/api/v1/common_pb.js";
import type {
  SessionSummary,
  SessionSummaryDetail,
  SessionBlock,
  SessionDetail,
  SessionEvent,
  SessionAttachment,
  Turn,
  ToolCall,
  ToolCallBadge as UiToolCallBadge,
} from "../types";

export function decodeSessionSummaries(response: unknown): SessionSummary[] {
  const decoded = fromJson(ListSessionsResponseSchema, response as never);
  return decoded.sessions.map(adaptSessionSummary);
}

export function decodeSessionSummaryDetail(response: unknown): SessionSummaryDetail {
  const decoded = fromJson(GetSessionSummaryResponseSchema, response as never);
  return adaptSessionSummaryDetail(decoded.session);
}

export function decodeSessionBlocks(response: unknown): SessionBlock[] {
  const decoded = fromJson(GetSessionBlocksResponseSchema, response as never);
  return decoded.blocks.map(adaptSessionBlock);
}

export function decodeSessionDetail(response: unknown): SessionDetail {
  const decoded = fromJson(GetSessionDetailResponseSchema, response as never);
  return adaptSessionDetail(decoded.session);
}

function adaptSessionSummary(summary?: PbSessionSummary): SessionSummary {
  return {
    id: summary?.id ?? "",
    title: summary?.title ?? "",
    summary: summary?.summary ?? null,
    classification: summary?.classification ?? "",
    timing: {
      started_at: summary?.timing?.startedAt ?? "",
      ended_at: summary?.timing?.endedAt ?? null,
      duration_seconds: summary?.timing?.durationSeconds ?? 0,
      active_duration_seconds: summary?.timing?.activeDurationSeconds ?? 0,
      hour_of_day: summary?.timing?.hourOfDay ?? 0,
      day_of_week: summary?.timing?.dayOfWeek ?? 0,
    },
    metrics: {
      turn_count: summary?.metrics?.turnCount ?? 0,
      tool_call_count: summary?.metrics?.toolCallCount ?? 0,
      ...activityMetrics(summary?.metrics),
      total_input_tokens: summary?.metrics?.totalInputTokens,
      total_output_tokens: summary?.metrics?.totalOutputTokens,
      total_cache_read_tokens: summary?.metrics?.totalCacheReadTokens,
    },
    environment: {
      agent_framework: summary?.environment?.agentFramework ?? "",
      model: summary?.environment?.model ?? "",
    },
    operational_context: {
      working_directory: summary?.operationalContext?.workingDirectory ?? "",
      autonomy_level: summary?.operationalContext?.autonomyLevel,
      sandbox: summary?.operationalContext?.sandbox,
    },
  };
}

function adaptSessionSummaryDetail(summary?: PbSessionSummaryDetail): SessionSummaryDetail {
  return {
    id: summary?.id ?? "",
    title: summary?.title ?? "",
    summary: summary?.summary ?? null,
    classification: summary?.classification ?? "",
    timing: {
      started_at: summary?.timing?.startedAt ?? "",
      ended_at: summary?.timing?.endedAt ?? null,
      duration_seconds: summary?.timing?.durationSeconds ?? 0,
      active_duration_seconds: summary?.timing?.activeDurationSeconds ?? 0,
      hour_of_day: summary?.timing?.hourOfDay ?? 0,
      day_of_week: summary?.timing?.dayOfWeek ?? 0,
    },
    metrics: {
      turn_count: summary?.metrics?.turnCount ?? 0,
      tool_call_count: summary?.metrics?.toolCallCount ?? 0,
      ...activityMetrics(summary?.metrics),
      total_input_tokens: summary?.metrics?.totalInputTokens,
      total_output_tokens: summary?.metrics?.totalOutputTokens,
      total_cache_read_tokens: summary?.metrics?.totalCacheReadTokens,
    },
    environment: {
      agent_framework: summary?.environment?.agentFramework ?? "",
      model: summary?.environment?.model ?? "",
    },
    operational_context: {
      working_directory: summary?.operationalContext?.workingDirectory ?? "",
      autonomy_level: summary?.operationalContext?.autonomyLevel,
      sandbox: summary?.operationalContext?.sandbox,
    },
    provenance: {
      source_format: summary?.provenance?.sourceFormat ?? "",
      source_path: summary?.provenance?.sourcePath ?? "",
      original_session_id: summary?.provenance?.originalSessionId ?? "",
      converted_at: summary?.provenance?.convertedAt ?? "",
    },
    events: summary?.events.map(adaptSessionEvent) ?? [],
    attachments: summary?.attachments.map(adaptSessionAttachment) ?? [],
  };
}

function adaptSessionDetail(detail?: PbSessionDetail): SessionDetail {
  return {
    id: detail?.id ?? "",
    title: detail?.title ?? "",
    summary: detail?.summary ?? null,
    classification: detail?.classification ?? "",
    timing: {
      started_at: detail?.timing?.startedAt ?? "",
      ended_at: detail?.timing?.endedAt ?? null,
      duration_seconds: detail?.timing?.durationSeconds ?? 0,
      active_duration_seconds: detail?.timing?.activeDurationSeconds ?? 0,
      hour_of_day: detail?.timing?.hourOfDay ?? 0,
      day_of_week: detail?.timing?.dayOfWeek ?? 0,
    },
    metrics: {
      turn_count: detail?.metrics?.turnCount ?? 0,
      tool_call_count: detail?.metrics?.toolCallCount ?? 0,
      ...activityMetrics(detail?.metrics),
      total_input_tokens: detail?.metrics?.totalInputTokens,
      total_output_tokens: detail?.metrics?.totalOutputTokens,
      total_cache_read_tokens: detail?.metrics?.totalCacheReadTokens,
    },
    environment: {
      agent_framework: detail?.environment?.agentFramework ?? "",
      model: detail?.environment?.model ?? "",
    },
    operational_context: {
      working_directory: detail?.operationalContext?.workingDirectory ?? "",
      autonomy_level: detail?.operationalContext?.autonomyLevel,
      sandbox: detail?.operationalContext?.sandbox,
    },
    provenance: {
      source_format: detail?.provenance?.sourceFormat ?? "",
      source_path: detail?.provenance?.sourcePath ?? "",
      original_session_id: detail?.provenance?.originalSessionId ?? "",
      converted_at: detail?.provenance?.convertedAt ?? "",
    },
    blocks: detail?.blocks.map(adaptSessionBlock) ?? [],
    unassociated_tool_calls: detail?.unassociatedToolCalls.map(adaptToolCall) ?? [],
    events: detail?.events.map(adaptSessionEvent) ?? [],
    attachments: detail?.attachments.map(adaptSessionAttachment) ?? [],
  };
}

function adaptSessionEvent(event: PbSessionEvent): SessionEvent {
  return {
    id: event.id,
    timestamp: event.timestamp,
    turn_index: event.turnIndex ?? null,
    ordinal: event.ordinal ?? null,
    kind: event.kind,
    role: event.role,
    tool_call_id: event.toolCallId ?? null,
    annotation_id: event.annotationId ?? null,
    attachment_id: event.attachmentId ?? null,
    title: event.title,
    summary: event.summary,
    text: event.text,
    severity: event.severity,
    collapsed_by_default: event.collapsedByDefault,
    framework_metadata: event.frameworkMetadata ?? null,
  };
}

function adaptSessionAttachment(attachment: PbSessionAttachment): SessionAttachment {
  return {
    id: attachment.id,
    timestamp: attachment.timestamp,
    kind: attachment.kind,
    name: attachment.name,
    media_type: attachment.mediaType,
    path: attachment.path,
    url: attachment.url,
    size_bytes: attachment.sizeBytes ?? null,
    hash: attachment.hash,
    content_ref: attachment.contentRef,
    text_preview: attachment.textPreview,
    turn_index: attachment.turnIndex ?? null,
    tool_call_id: attachment.toolCallId ?? null,
    event_id: attachment.eventId ?? null,
    framework_metadata: attachment.frameworkMetadata ?? null,
  };
}

function adaptSessionBlock(block: PbSessionBlock): SessionBlock {
  return {
    block_num: block.blockNum,
    user_turn_idx: block.userTurnIdx,
    user_ts: block.userTs,
    user_content: block.userContent,
    agent_turns: block.agentTurns,
    tool_calls: block.toolCalls,
    gap_minutes: block.gapMinutes ?? null,
    turns: block.turns.map(adaptTurn),
    artifacts: {
      commits: [...(block.artifacts?.commits ?? [])],
      tickets_created: [...(block.artifacts?.ticketsCreated ?? [])],
      docs_added: [...(block.artifacts?.docsAdded ?? [])],
      diary_writes: block.artifacts?.diaryWrites ?? 0,
    },
  };
}

function adaptTurn(turn: PbTurn): Turn {
  return {
    idx: turn.idx,
    role: turn.role as Turn["role"],
    source: turn.source,
    content: turn.content,
    timestamp: turn.timestamp,
    thinking: turn.thinking ?? null,
    model: turn.model ?? null,
    usage: turn.usage
      ? {
          input_tokens: turn.usage.inputTokens ?? null,
          output_tokens: turn.usage.outputTokens ?? null,
          cache_read_tokens: turn.usage.cacheReadTokens ?? null,
          reasoning_tokens: turn.usage.reasoningTokens ?? null,
        }
      : null,
    tool_calls_in_turn: turn.toolCallsInTurn.map(adaptToolCall),
  };
}

function adaptToolCall(toolCall: PbToolCall): ToolCall {
  return {
    id: toolCall.id,
    tool_name: toolCall.toolName,
    timestamp: toolCall.timestamp,
    operation_type: toolCall.operationType,
    record_kind: toolCall.recordKind,
    framework_metadata: toolCall.frameworkMetadata,
    input: {
      command: toolCall.input?.command,
      arguments: toolCall.input?.arguments,
      file_path: toolCall.input?.filePath ?? null,
      file_targets: toolCall.input?.fileTargets.map((target) => ({
        path: target.path, native_path: target.nativePath,
        operation_type: target.operationType, evidence_kind: target.evidenceKind,
        status: target.status, success: target.success ?? null,
        cwd: target.cwd, resolved: target.resolved, source_reference: target.sourceReference,
      })) ?? [],
    },
    output: {
      success: toolCall.output?.success ?? null,
      status: toolCall.output?.status,
      exit_code: toolCall.output?.exitCode ?? null,
      result: toolCall.output?.result ?? null,
      error: toolCall.output?.error ?? null,
      duration_ms: toolCall.output?.durationMs ?? 0,
      truncated: toolCall.output?.truncated ?? false,
      full_reference: toolCall.output?.fullReference ?? null,
      full_bytes: toolCall.output?.fullBytes == null ? null : Number(toolCall.output.fullBytes),
      full_hash: toolCall.output?.fullHash ?? null,
    },
    badges: toolCall.badges.map(adaptToolCallBadge),
  };
}

function adaptToolCallBadge(badge: ToolCallBadge): UiToolCallBadge {
  switch (badge) {
    case ToolCallBadge.COMMIT:
      return "commit";
    case ToolCallBadge.TICKET_CREATE:
      return "ticket-create";
    case ToolCallBadge.DOC_ADD:
      return "doc-add";
    case ToolCallBadge.DIARY_WRITE:
      return "diary-write";
    case ToolCallBadge.ERROR:
      return "error";
    case ToolCallBadge.UNSPECIFIED:
    default:
      return "error";
  }
}
