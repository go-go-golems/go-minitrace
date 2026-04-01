import { http, HttpResponse } from "msw";
import {
  mockSessions,
  mockSessionDetail,
  mockPresets,
  mockSavedQueries,
  mockQueryResult,
} from "./data";

export const handlers = [
  http.get("/api/sessions", () => {
    return HttpResponse.json(mockSessions);
  }),

  http.get("/api/sessions/:id", ({ params }) => {
    if (params.id === mockSessionDetail.id) {
      return HttpResponse.json(mockSessionDetail);
    }
    return HttpResponse.json({ error: "not found" }, { status: 404 });
  }),

  http.get("/api/sessions/:id/blocks", ({ params }) => {
    if (params.id === mockSessionDetail.id) {
      return HttpResponse.json(mockSessionDetail.blocks);
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

  http.get("/api/queries", () => {
    return HttpResponse.json(mockSavedQueries);
  }),

  http.post("/api/queries", async ({ request }) => {
    const body = (await request.json()) as { name: string; sql: string };
    return HttpResponse.json({ ...body, path: `my-queries/${body.name}.sql` }, { status: 201 });
  }),
];
