/** Core minitrace session types matching the DuckDB schema */

export interface SessionTiming {
  started_at: string;
  ended_at: string | null;
  duration_seconds: number;
  active_duration_seconds: number;
  hour_of_day: number;
  day_of_week: number;
}

export interface SessionMetrics {
  turn_count: number;
  tool_call_count: number;
  total_input_tokens?: number;
  total_output_tokens?: number;
  total_cache_read_tokens?: number;
}

export interface SessionEnvironment {
  agent_framework: string;
  model: string;
}

export interface SessionOperationalContext {
  working_directory: string;
  autonomy_level?: string;
  sandbox?: boolean;
}

export interface SessionProvenance {
  source_format: string;
  source_path: string;
  original_session_id: string;
  converted_at: string;
}

/** Summary row returned by /api/sessions */
export interface SessionSummary {
  id: string;
  title: string;
  summary: string | null;
  classification: string;
  timing: SessionTiming;
  metrics: SessionMetrics;
  environment: SessionEnvironment;
  operational_context: SessionOperationalContext;
}

/** Tool call within a turn */
export interface ToolCall {
  id: string;
  tool_name: string;
  timestamp: string;
  operation_type: string;
  input: {
    command?: string;
    arguments?: Record<string, unknown>;
    file_path?: string | null;
  };
  output: {
    success: boolean;
    result: string | null;
    error: string | null;
    duration_ms: number;
    truncated: boolean;
  };
  badges: ToolCallBadge[];
}

export type ToolCallBadge =
  | "commit"
  | "ticket-create"
  | "doc-add"
  | "diary-write"
  | "error";

/** A single conversation turn */
export interface Turn {
  idx: number;
  role: "user" | "assistant" | "system";
  source: string;
  content: string;
  timestamp: string;
  tool_calls_in_turn: ToolCall[];
}

/** Artifact summary for a block */
export interface BlockArtifacts {
  commits: string[];
  tickets_created: string[];
  docs_added: string[];
  diary_writes: number;
}

/** A human-input block: one user prompt + all agent response until next user prompt */
export interface SessionBlock {
  block_num: number;
  user_turn_idx: number;
  user_ts: string;
  user_content: string;
  agent_turns: number;
  tool_calls: number;
  gap_minutes: number | null;
  turns: Turn[];
  artifacts: BlockArtifacts;
}

/** Full session detail returned by /api/sessions/:id */
export interface SessionDetail {
  id: string;
  title: string;
  summary: string | null;
  classification: string;
  timing: SessionTiming;
  metrics: SessionMetrics;
  environment: SessionEnvironment;
  operational_context: SessionOperationalContext;
  provenance: SessionProvenance;
  blocks: SessionBlock[];
}
