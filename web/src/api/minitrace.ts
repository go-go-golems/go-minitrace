import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import type {
  SessionSummary,
  SessionDetail,
  SessionBlock,
  SavedQuery,
  QueryResult,
} from "../types";

export const minitraceApi = createApi({
  reducerPath: "minitraceApi",
  baseQuery: fetchBaseQuery({ baseUrl: "/api" }),
  tagTypes: ["Sessions", "Queries"],
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
} = minitraceApi;
