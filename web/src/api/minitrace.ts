import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type {
  SessionSummary,
  SessionDetail,
  SessionBlock,
  SavedQuery,
  QueryResult,
  Annotation,
  SessionAnnotationsResponse,
  SyncReport,
} from "../types";

export const minitraceApi = createApi({
  reducerPath: "minitraceApi",
  baseQuery: fetchBaseQuery({ baseUrl: "/api" }),
  tagTypes: ["Sessions", "Queries", "Annotations"],
  endpoints: (builder) => ({
    // ── sessions ─────────────────────────────────
    getSessions: builder.query<SessionSummary[], void>({
      query: () => "sessions",
      providesTags: ["Sessions"],
    }),

    getSession: builder.query<SessionDetail, string>({
      query: (id) => `sessions/${id}`,
    }),

    getSessionBlocks: builder.query<SessionBlock[], string>({
      query: (id) => `sessions/${id}/blocks`,
    }),

    // ── queries ──────────────────────────────────
    executeQuery: builder.mutation<QueryResult, { sql: string }>({
      query: (body) => ({ url: "query", method: "POST", body }),
    }),

    getPresets: builder.query<SavedQuery[], void>({
      query: () => "presets",
    }),

    getSavedQueries: builder.query<SavedQuery[], void>({
      query: () => "queries",
      providesTags: ["Queries"],
    }),

    saveQuery: builder.mutation<
      SavedQuery,
      { name: string; folder: string; description: string; sql: string }
    >({
      query: (body) => ({ url: "queries", method: "POST", body }),
      invalidatesTags: ["Queries"],
    }),

    // ── annotations ────────────────────────────────
    getSessionAnnotations: builder.query<SessionAnnotationsResponse, string>({
      query: (sessionId) => `sessions/${sessionId}/annotations`,
      providesTags: (_result, _error, sessionId) => [
        { type: "Annotations", id: sessionId },
      ],
    }),

    createAnnotation: builder.mutation<
      Annotation,
      { session_id: string; category: string; title: string; detail?: string; scope_type?: string; target_id?: string; tags?: string[] }
    >({
      query: ({ session_id, ...body }) => ({
        url: `sessions/${session_id}/annotations`,
        method: "POST",
        body,
      }),
      invalidatesTags: (_result, _error, { session_id }) => [
        { type: "Annotations", id: session_id },
      ],
    }),

    updateAnnotation: builder.mutation<
      { id: string; status: string },
      { id: string; patch: Record<string, unknown> }
    >({
      query: ({ id, patch }) => ({
        url: `annotations/${id}`,
        method: "PUT",
        body: patch,
      }),
      invalidatesTags: (_result, _error, { id: _annotationId }) => [
        { type: "Annotations" },
      ],
    }),

    deleteAnnotation: builder.mutation<void, { id: string; session_id: string }>({
      query: ({ id }) => ({ url: `annotations/${id}`, method: "DELETE" }),
      invalidatesTags: (_result, _error, { session_id }) => [
        { type: "Annotations", id: session_id },
      ],
    }),

    syncAnnotations: builder.mutation<SyncReport, { session_id?: string; dry_run?: boolean }>({
      query: (body = {}) => ({ url: "annotations/sync", method: "POST", body }),
      invalidatesTags: [{ type: "Annotations" }],
    }),
  }),
});

export const {
  useGetSessionsQuery,
  useGetSessionQuery,
  useGetSessionBlocksQuery,
  useExecuteQueryMutation,
  useGetPresetsQuery,
  useGetSavedQueriesQuery,
  useSaveQueryMutation,
  useGetSessionAnnotationsQuery,
  useCreateAnnotationMutation,
  useUpdateAnnotationMutation,
  useDeleteAnnotationMutation,
  useSyncAnnotationsMutation,
} = minitraceApi;
