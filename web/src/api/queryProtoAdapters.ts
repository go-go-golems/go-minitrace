import { fromJson } from "@bufbuild/protobuf";
import {
  ListPresetsResponseSchema,
  ListQueriesResponseSchema,
  SavedQuerySchema,
  type SavedQuery as PbSavedQuery,
} from "../gen/proto/go_go_golems/minitrace/api/v1/queries_pb.js";
import type { SavedQuery } from "../types";

export function decodePresets(response: unknown): SavedQuery[] {
  const decoded = fromJson(ListPresetsResponseSchema, response as never);
  return decoded.presets.map(adaptSavedQuery);
}

export function decodeSavedQueries(response: unknown): SavedQuery[] {
  const decoded = fromJson(ListQueriesResponseSchema, response as never);
  return decoded.queries.map(adaptSavedQuery);
}

export function decodeSavedQuery(response: unknown): SavedQuery {
  const decoded = fromJson(SavedQuerySchema, response as never);
  return adaptSavedQuery(decoded);
}

function adaptSavedQuery(query: PbSavedQuery): SavedQuery {
  return {
    name: query.name,
    folder: query.folder,
    path: query.path,
    description: query.description,
    sql: query.sql,
    readonly: query.readonly,
  };
}
