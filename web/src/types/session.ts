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

/** Summary detail returned by /api/sessions/:id/summary */
export interface SessionSummaryDetail {
  id: string;
  title: string;
  summary: string | null;
  classification: string;
  timing: SessionTiming;
  metrics: SessionMetrics;
  environment: SessionEnvironment;
  operational_context: SessionOperationalContext;
  provenance: SessionProvenance;
}

/** Full session detail returned by /api/sessions/:id */
export interface SessionDetail extends SessionSummaryDetail {
  blocks: SessionBlock[];
}

/** Annotation types — matches the minitrace annotation schema */

// A single annotation on a session, turn, or tool_call.
export interface Annotation {
  id: string;
  timestamp: string;
  annotator: string;
  scope: {
    type: "session" | "turn" | "tool_call";
    target_id: string;
  };
  content: {
    category: string;
    tags: string[];
    title: string;
    detail: string;
  };
  taxonomy_mappings: {
    minitrace: string[];
    mast: string[];
    toolemu: string[];
  };
  classification?: string;
}

export type AnnotationCategory =
  | "observation"
  | "ai-failure"
  | "user-error"
  | "environment-issue"
  | "success"
  | "question"
  | "to-discuss"
  | "to-improve";

export const ANNOTATION_CATEGORY_COLORS: Record<string, "default" | "primary" | "secondary" | "error" | "info" | "success" | "warning"> = {
  observation: "default",
  "ai-failure": "error",
  "user-error": "error",
  "environment-issue": "warning",
  success: "success",
  question: "info",
  "to-discuss": "secondary",
  "to-improve": "primary",
};

// Response from GET /api/sessions/:id/annotations
export interface SessionAnnotationsResponse {
  session_id: string;
  count: number;
  annotations: Annotation[];
}

// Row from GET /api/v2/annotations (intentional API schema, not Go-exported field casing)
export interface AnnotationListRow {
  id: string;
  sessionId: string;
  annotator: string;
  scopeType: string;
  targetId: string;
  category: string;
  title: string;
  detail: string;
  tags: string[];
  taxonomyMinitrace: string[];
  taxonomyMast: string[];
  taxonomyToolemu: string[];
  classification?: string | null;
  createdAt: string;
  updatedAt: string;
}

// Sync report returned by POST /api/annotations/sync
export interface SyncReport {
  synced: string[];
  skipped: string[];
  errors: { session_id: string; error: string }[];
}
