import { fromJson } from "@bufbuild/protobuf";
import {
  ExecuteQueryCommandResponseSchema,
  ListQueryCommandsResponseSchema,
  QueryCommandKind,
  type QueryCommand as PbQueryCommand,
  type QueryCommandParam as PbQueryCommandParam,
} from "../gen/proto/go_go_golems/minitrace/api/v1/query_commands_pb.js";
import type { QueryCommand, QueryCommandKind as UIQueryCommandKind, QueryCommandParam, QueryResult } from "../types";

export function decodeQueryCommands(response: unknown): QueryCommand[] {
  const decoded = fromJson(ListQueryCommandsResponseSchema, response as never);
  return decoded.commands.map(adaptQueryCommand);
}

export function decodeExecuteQueryCommandResponse(response: unknown): QueryResult {
  const decoded = fromJson(ExecuteQueryCommandResponseSchema, response as never);
  return {
    columns: decoded.columns,
    rows: decoded.rows as Record<string, unknown>[],
    duration_ms: Number(decoded.durationMs),
    row_count: decoded.rowCount,
    rendered_sql: decoded.renderedSql,
  };
}

export function buildExecuteQueryCommandBody(values: Record<string, unknown>, renderOnly = false) {
  return {
    values,
    renderOnly,
  };
}

function adaptQueryCommand(command: PbQueryCommand): QueryCommand {
  return {
    name: command.name,
    folder: command.folder,
    path: command.path,
    shortDescription: command.shortDescription,
    longDescription: command.longDescription,
    flags: command.flags.map((flag) => adaptQueryCommandParam(flag)),
    arguments: command.arguments.map((argument) => adaptQueryCommandParam(argument)),
    tags: command.tags,
    readonly: command.readonly,
    kind: adaptQueryCommandKind(command.kind),
    aliasFor: command.aliasFor,
  };
}

function adaptQueryCommandParam(param: PbQueryCommandParam): QueryCommandParam {
  return {
    name: param.name,
    type: param.type,
    help: param.help,
    required: param.required,
    defaultJson: param.defaultJson,
    choices: param.choices,
    positional: param.positional,
    shortFlag: param.shortFlag,
  };
}

function adaptQueryCommandKind(kind: QueryCommandKind): UIQueryCommandKind {
  switch (kind) {
    case QueryCommandKind.ALIAS:
      return "alias";
    case QueryCommandKind.VERB:
    case QueryCommandKind.UNSPECIFIED:
    default:
      return "verb";
  }
}
