import { fromJson } from "@bufbuild/protobuf";
import {
  AnnotationCategory,
  AnnotationScopeType,
  AnnotationSchema,
  GetSessionAnnotationsResponseSchema,
  ListAnnotationsResponseSchema,
  SyncAnnotationsResponseSchema,
  UpdateAnnotationResponseSchema,
  type Annotation as PbAnnotation,
} from "../gen/proto/go_go_golems/minitrace/api/v1/annotations_pb.js";
import type {
  Annotation,
  AnnotationListRow,
  SessionAnnotationsResponse,
  SyncReport,
} from "../types";

const protoCategoryToUi: Record<number, string> = {
  [AnnotationCategory.OBSERVATION]: "observation",
  [AnnotationCategory.AI_FAILURE]: "ai-failure",
  [AnnotationCategory.USER_ERROR]: "user-error",
  [AnnotationCategory.ENVIRONMENT_ISSUE]: "environment-issue",
  [AnnotationCategory.SUCCESS]: "success",
  [AnnotationCategory.QUESTION]: "question",
  [AnnotationCategory.TO_DISCUSS]: "to-discuss",
  [AnnotationCategory.TO_IMPROVE]: "to-improve",
};

const uiCategoryToProto: Record<string, number> = {
  observation: AnnotationCategory.OBSERVATION,
  "ai-failure": AnnotationCategory.AI_FAILURE,
  "user-error": AnnotationCategory.USER_ERROR,
  "environment-issue": AnnotationCategory.ENVIRONMENT_ISSUE,
  success: AnnotationCategory.SUCCESS,
  question: AnnotationCategory.QUESTION,
  "to-discuss": AnnotationCategory.TO_DISCUSS,
  "to-improve": AnnotationCategory.TO_IMPROVE,
};

const protoScopeToUi: Record<number, string> = {
  [AnnotationScopeType.SESSION]: "session",
  [AnnotationScopeType.TURN]: "turn",
  [AnnotationScopeType.TOOL_CALL]: "tool_call",
};

const uiScopeToProto: Record<string, number> = {
  session: AnnotationScopeType.SESSION,
  turn: AnnotationScopeType.TURN,
  tool_call: AnnotationScopeType.TOOL_CALL,
};

export function decodeAnnotation(response: unknown): Annotation {
  const decoded = fromJson(AnnotationSchema, response as never);
  return adaptAnnotation(decoded);
}

export function decodeSessionAnnotations(response: unknown): SessionAnnotationsResponse {
  const decoded = fromJson(GetSessionAnnotationsResponseSchema, response as never);
  return {
    session_id: decoded.sessionId,
    count: decoded.count,
    annotations: decoded.annotations.map(adaptAnnotation),
  };
}

export function decodeAnnotationRows(response: unknown): AnnotationListRow[] {
  const decoded = fromJson(ListAnnotationsResponseSchema, response as never);
  return decoded.annotations.map((row) => ({
    id: row.id,
    sessionId: row.sessionId,
    annotator: row.annotator,
    scopeType: protoScopeToUi[row.scopeType] ?? "session",
    targetId: row.targetId,
    category: protoCategoryToUi[row.category] ?? "observation",
    title: row.title,
    detail: row.detail,
    tags: [...row.tags],
    taxonomyMinitrace: [...row.taxonomyMinitrace],
    taxonomyMast: [...row.taxonomyMast],
    taxonomyToolemu: [...row.taxonomyToolemu],
    classification: row.classification ?? null,
    createdAt: row.createdAt,
    updatedAt: row.updatedAt,
  }));
}

export function decodeUpdateStatus(response: unknown): { id: string; status: string } {
  const decoded = fromJson(UpdateAnnotationResponseSchema, response as never);
  return {
    id: decoded.id,
    status: decoded.status,
  };
}

export function decodeSyncReport(response: unknown): SyncReport {
  const decoded = fromJson(SyncAnnotationsResponseSchema, response as never);
  return {
    synced: [...decoded.synced],
    skipped: [...decoded.skipped],
    errors: decoded.errors.map((err) => ({
      session_id: err.sessionId,
      error: err.error,
    })),
  };
}

export function buildCreateAnnotationBody(args: {
  annotator?: string;
  scope_type?: string;
  target_id?: string;
  category: string;
  title: string;
  detail?: string;
  tags?: string[];
}): Record<string, unknown> {
  return {
    annotator: args.annotator,
    scopeType: uiScopeToProto[args.scope_type ?? "session"] ?? AnnotationScopeType.SESSION,
    targetId: args.target_id ?? "",
    category: uiCategoryToProto[args.category] ?? AnnotationCategory.UNSPECIFIED,
    title: args.title,
    detail: args.detail ?? "",
    tags: args.tags ?? [],
  };
}

export function buildUpdateAnnotationBody(patch: Record<string, unknown>): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (typeof patch.title === "string") {
    body.title = patch.title;
  }
  if (typeof patch.detail === "string") {
    body.detail = patch.detail;
  }
  if (typeof patch.category === "string") {
    body.category = uiCategoryToProto[patch.category] ?? AnnotationCategory.UNSPECIFIED;
  }
  if (Array.isArray(patch.tags)) {
    body.tags = { values: patch.tags };
  }
  if (Array.isArray(patch.taxonomy_minitrace)) {
    body.taxonomyMinitrace = { values: patch.taxonomy_minitrace };
  }
  if (Array.isArray(patch.taxonomy_mast)) {
    body.taxonomyMast = { values: patch.taxonomy_mast };
  }
  if (Array.isArray(patch.taxonomy_toolemu)) {
    body.taxonomyToolemu = { values: patch.taxonomy_toolemu };
  }
  if (typeof patch.classification === "string") {
    body.classification = patch.classification;
  }
  return body;
}

export function buildSyncAnnotationsBody(args: { session_id?: string; dry_run?: boolean }): Record<string, unknown> {
  return {
    sessionId: args.session_id,
    dryRun: args.dry_run ?? false,
  };
}

function adaptAnnotation(annotation: PbAnnotation): Annotation {
  return {
    id: annotation.id,
    timestamp: annotation.timestamp,
    annotator: annotation.annotator,
    scope: {
      type: (protoScopeToUi[annotation.scope?.type ?? AnnotationScopeType.SESSION] ?? "session") as Annotation["scope"]["type"],
      target_id: annotation.scope?.targetId ?? "",
    },
    content: {
      category: protoCategoryToUi[annotation.content?.category ?? AnnotationCategory.OBSERVATION] ?? "observation",
      tags: [...(annotation.content?.tags ?? [])],
      title: annotation.content?.title ?? "",
      detail: annotation.content?.detail ?? "",
    },
    taxonomy_mappings: {
      minitrace: [...(annotation.taxonomyMappings?.minitrace ?? [])],
      mast: [...(annotation.taxonomyMappings?.mast ?? [])],
      toolemu: [...(annotation.taxonomyMappings?.toolemu ?? [])],
    },
    classification: annotation.classification,
  };
}
