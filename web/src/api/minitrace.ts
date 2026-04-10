import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type {
  SessionSummary,
  SessionSummaryDetail,
  SessionDetail,
  SessionBlock,
  SavedQuery,
  QueryCommand,
  QueryResult,
  Annotation,
  AnnotationListRow,
  SessionAnnotationsResponse,
  SyncReport,
} from "../types";
import {
  decodeSessionBlocks,
  decodeSessionDetail,
  decodeSessionSummaries,
  decodeSessionSummaryDetail,
} from "./sessionProtoAdapters";
import {
  buildCreateAnnotationBody,
  buildSyncAnnotationsBody,
  buildUpdateAnnotationBody,
  decodeAnnotation,
  decodeAnnotationRows,
  decodeSessionAnnotations,
  decodeSyncReport,
  decodeUpdateStatus,
} from "./annotationProtoAdapters";
import {
  decodePresets,
  decodeSavedQueries,
  decodeSavedQuery,
} from "./queryProtoAdapters";
import {
  buildExecuteQueryCommandBody,
  decodeExecuteQueryCommandResponse,
  decodeQueryCommands,
} from "./queryCommandProtoAdapters";

function encodePathSegments(path: string): string {
  return path
    .split("/")
    .map((segment) => encodeURIComponent(segment))
    .join("/");
}

export const minitraceApi = createApi({
  reducerPath: "minitraceApi",
  baseQuery: fetchBaseQuery({ baseUrl: "/api" }),
  tagTypes: ["Sessions", "Queries", "QueryCommands", "Annotations"],
  endpoints: (builder) => ({
    // ── sessions ─────────────────────────────────
    getSessions: builder.query<SessionSummary[], void>({
      query: () => "v2/sessions",
      transformResponse: decodeSessionSummaries,
      providesTags: ["Sessions"],
    }),

    getSession: builder.query<SessionDetail, string>({
      query: (id) => `v2/sessions/${id}`,
      transformResponse: decodeSessionDetail,
    }),

    getSessionSummary: builder.query<SessionSummaryDetail, string>({
      query: (id) => `v2/sessions/${id}/summary`,
      transformResponse: decodeSessionSummaryDetail,
    }),

    getSessionBlocks: builder.query<SessionBlock[], string>({
      query: (id) => `v2/sessions/${id}/blocks`,
      transformResponse: decodeSessionBlocks,
    }),

    // ── queries ──────────────────────────────────
    executeQuery: builder.mutation<QueryResult, { sql: string }>({
      query: (body) => ({ url: "query", method: "POST", body }),
    }),

    getPresets: builder.query<SavedQuery[], void>({
      query: () => "v2/presets",
      transformResponse: decodePresets,
    }),

    getSavedQueries: builder.query<SavedQuery[], void>({
      query: () => "v2/queries",
      transformResponse: decodeSavedQueries,
      providesTags: ["Queries"],
    }),

    getQueryCommands: builder.query<QueryCommand[], void>({
      query: () => "v2/query-commands",
      transformResponse: decodeQueryCommands,
      providesTags: ["QueryCommands"],
    }),

    executeQueryCommand: builder.mutation<
      QueryResult,
      { path: string; values: Record<string, unknown>; renderOnly?: boolean }
    >({
      query: ({ path, values, renderOnly = false }) => ({
        url: `v2/query-commands/${encodePathSegments(path)}/execute`,
        method: "POST",
        body: buildExecuteQueryCommandBody(values, renderOnly),
      }),
      transformResponse: decodeExecuteQueryCommandResponse,
    }),

    saveQuery: builder.mutation<
      SavedQuery,
      { name: string; folder: string; description: string; sql: string }
    >({
      query: (body) => ({ url: "v2/queries", method: "POST", body }),
      transformResponse: decodeSavedQuery,
      invalidatesTags: ["Queries"],
    }),

    // ── annotations ────────────────────────────────
    getSessionAnnotations: builder.query<SessionAnnotationsResponse, string>({
      query: (sessionId) => `v2/sessions/${sessionId}/annotations`,
      transformResponse: decodeSessionAnnotations,
      providesTags: (_result, _error, sessionId) => [
        { type: "Annotations", id: sessionId },
      ],
    }),

    getAnnotations: builder.query<AnnotationListRow[], void>({
      query: () => "v2/annotations",
      transformResponse: decodeAnnotationRows,
      providesTags: [{ type: "Annotations" }],
    }),

    createAnnotation: builder.mutation<
      Annotation,
      { session_id: string; category: string; title: string; detail?: string; scope_type?: string; target_id?: string; tags?: string[] }
    >({
      query: ({ session_id, ...body }) => ({
        url: `v2/sessions/${session_id}/annotations`,
        method: "POST",
        body: buildCreateAnnotationBody(body),
      }),
      transformResponse: decodeAnnotation,
      invalidatesTags: (_result, _error, { session_id }) => [
        { type: "Annotations", id: session_id },
      ],
    }),

    updateAnnotation: builder.mutation<
      { id: string; status: string },
      { id: string; patch: Record<string, unknown> }
    >({
      query: ({ id, patch }) => ({
        url: `v2/annotations/${id}`,
        method: "PUT",
        body: buildUpdateAnnotationBody(patch),
      }),
      transformResponse: decodeUpdateStatus,
      invalidatesTags: [{ type: "Annotations" }],
    }),

    deleteAnnotation: builder.mutation<void, { id: string; session_id: string }>({
      query: ({ id }) => ({ url: `v2/annotations/${id}`, method: "DELETE" }),
      transformResponse: () => undefined,
      invalidatesTags: (_result, _error, { session_id }) => [
        { type: "Annotations", id: session_id },
      ],
    }),

    syncAnnotations: builder.mutation<SyncReport, { session_id?: string; dry_run?: boolean }>({
      query: (body = {}) => ({ url: "v2/annotations/sync", method: "POST", body: buildSyncAnnotationsBody(body) }),
      transformResponse: decodeSyncReport,
      invalidatesTags: [{ type: "Annotations" }],
    }),
  }),
});

export const {
  useGetSessionsQuery,
  useGetSessionQuery,
  useGetSessionSummaryQuery,
  useGetSessionBlocksQuery,
  useExecuteQueryMutation,
  useGetPresetsQuery,
  useGetSavedQueriesQuery,
  useGetQueryCommandsQuery,
  useExecuteQueryCommandMutation,
  useSaveQueryMutation,
  useGetSessionAnnotationsQuery,
  useGetAnnotationsQuery,
  useCreateAnnotationMutation,
  useUpdateAnnotationMutation,
  useDeleteAnnotationMutation,
  useSyncAnnotationsMutation,
} = minitraceApi;
