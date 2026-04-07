import { http, HttpResponse } from "msw";
import {
  mockSessions,
  mockSessionDetail,
  mockPresets,
  mockSavedQueries,
  mockQueryResult,
} from "./data";

function buildMockSessionDetail(id: string) {
  const summary = mockSessions.find((session) => session.id === id);
  if (!summary) {
    return null;
  }

  return {
    ...mockSessionDetail,
    ...summary,
    provenance: mockSessionDetail.provenance,
    blocks: mockSessionDetail.blocks,
  };
}

export const handlers = [
  http.get("/api/sessions", () => {
    return HttpResponse.json(mockSessions);
  }),

  http.get("/api/sessions/:id", ({ params }) => {
    const sessionDetail = buildMockSessionDetail(String(params.id ?? ""));
    if (sessionDetail) {
      return HttpResponse.json(sessionDetail);
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),

  http.get("/api/sessions/:id/blocks", ({ params }) => {
    const sessionDetail = buildMockSessionDetail(String(params.id ?? ""));
    if (sessionDetail) {
      return HttpResponse.json(sessionDetail.blocks);
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),

  http.post("/api/query", async ({ request }) => {
    const body = (await request.json()) as { sql: string };
    if (body.sql.toLowerCase().includes("error")) {
      return HttpResponse.json(
        {
          columns: [],
          rows: [],
          duration_ms: 3,
          row_count: 0,
          error: { message: 'Binder Error: Referenced column "error" not found' },
        },
        { status: 400 }
      );
    }
    return HttpResponse.json(mockQueryResult);
  }),

  http.get("/api/presets", () => {
    return HttpResponse.json(mockPresets);
  }),

  http.get("/api/v2/presets", () => {
    return HttpResponse.json({ meta: { schemaVersion: 1 }, presets: mockPresets });
  }),

  http.get("/api/queries", () => {
    return HttpResponse.json(mockSavedQueries);
  }),

  http.get("/api/v2/queries", () => {
    return HttpResponse.json({ meta: { schemaVersion: 1 }, queries: mockSavedQueries });
  }),

  http.post("/api/queries", async ({ request }) => {
    const body = (await request.json()) as { name: string; sql: string; folder?: string; description?: string };
    return HttpResponse.json({ ...body, path: `my-queries/${body.name}.sql` }, { status: 201 });
  }),

  http.post("/api/v2/queries", async ({ request }) => {
    const body = (await request.json()) as { name: string; sql: string; folder?: string; description?: string };
    return HttpResponse.json(
      {
        name: body.name,
        folder: body.folder ?? "my-queries",
        path: `my-queries/${body.name}.sql`,
        description: body.description ?? "",
        sql: body.sql,
        readonly: false,
      },
      { status: 201 }
    );
  }),
];
