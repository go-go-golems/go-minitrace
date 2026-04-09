import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useSelector, useDispatch } from "react-redux";
import {
  useExecuteQueryMutation,
  useExecuteQueryCommandMutation,
  useGetPresetsQuery,
  useGetQueryCommandsQuery,
  useGetSavedQueriesQuery,
  useSaveQueryMutation,
} from "../api/minitrace";
import { QueryEditor } from "../components/QueryEditor";
import type { QueryCommand, QueryCommandParam, SavedQuery } from "../types";
import type { RootState, AppDispatch } from "../store";
import { setQuerySql, openQueryForSession } from "../store";

type QuerySourceKind = "preset" | "saved" | "command";

type ExecutionMode = "sql" | "command" | null;

interface ActiveQuerySource {
  kind: QuerySourceKind;
  path: string;
}

export function QueryEditorPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const dispatch = useDispatch<AppDispatch>();
  const { queryEditorSql } = useSelector((state: RootState) => state.ui);
  const [activeSource, setActiveSource] = useState<ActiveQuerySource | null>(null);
  const [activeCommandValues, setActiveCommandValues] = useState<Record<string, unknown>>({});
  const [executionMode, setExecutionMode] = useState<ExecutionMode>(null);
  const [lastLoadedSql, setLastLoadedSql] = useState<string | null>(null);
  const [externalUpdateAvailable, setExternalUpdateAvailable] = useState(false);

  const presetPollingInterval = activeSource?.kind === "preset" ? 3000 : 15000;
  const savedPollingInterval = activeSource?.kind === "saved" ? 3000 : 15000;

  const { data: presets = [] } = useGetPresetsQuery(undefined, {
    pollingInterval: presetPollingInterval,
    skipPollingIfUnfocused: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });
  const { data: savedQueries = [] } = useGetSavedQueriesQuery(undefined, {
    pollingInterval: savedPollingInterval,
    skipPollingIfUnfocused: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });
  const { data: queryCommands = [] } = useGetQueryCommandsQuery();
  const [executeQuery, sqlExecution] = useExecuteQueryMutation();
  const [executeQueryCommand, commandExecution] = useExecuteQueryCommandMutation();
  const [saveQuery] = useSaveQueryMutation();

  const activeSourceQuery = useMemo(() => {
    if (!activeSource || activeSource.kind === "command") {
      return null;
    }

    const queries = activeSource.kind === "preset" ? presets : savedQueries;
    return queries.find((query) => query.path === activeSource.path) ?? null;
  }, [activeSource, presets, savedQueries]);

  const activeCommand = useMemo(() => {
    if (!activeSource || activeSource.kind !== "command") {
      return null;
    }
    return queryCommands.find((command) => command.path === activeSource.path) ?? null;
  }, [activeSource, queryCommands]);

  useEffect(() => {
    const sessionId = searchParams.get("session");
    if (sessionId) {
      queueMicrotask(() => {
        setActiveSource(null);
        setActiveCommandValues({});
        setExecutionMode(null);
        setLastLoadedSql(null);
        setExternalUpdateAvailable(false);
      });
      dispatch(openQueryForSession(sessionId));
    }
  }, [searchParams, dispatch]);

  useEffect(() => {
    if (!activeSource || activeSource.kind === "command" || !activeSourceQuery) {
      return;
    }

    const latestSql = activeSourceQuery.sql;
    if (lastLoadedSql === null) {
      queueMicrotask(() => setLastLoadedSql(latestSql));
      return;
    }

    if (latestSql === lastLoadedSql) {
      if (queryEditorSql === latestSql && externalUpdateAvailable) {
        queueMicrotask(() => setExternalUpdateAvailable(false));
      }
      return;
    }

    if (queryEditorSql === lastLoadedSql) {
      dispatch(setQuerySql(latestSql));
      queueMicrotask(() => {
        setLastLoadedSql(latestSql);
        setExternalUpdateAvailable(false);
      });
      return;
    }

    queueMicrotask(() => setExternalUpdateAvailable(true));
  }, [
    activeSource,
    activeSourceQuery,
    dispatch,
    externalUpdateAvailable,
    lastLoadedSql,
    queryEditorSql,
  ]);

  const handleSelectQuery = (kind: "preset" | "saved") => (query: SavedQuery) => {
    setActiveSource({ kind, path: query.path });
    setActiveCommandValues({});
    setLastLoadedSql(query.sql);
    setExternalUpdateAvailable(false);
    dispatch(setQuerySql(query.sql));
  };

  const handleSelectCommand = (command: QueryCommand) => {
    setActiveSource({ kind: "command", path: command.path });
    setActiveCommandValues(buildInitialCommandValues(command));
    setLastLoadedSql(null);
    setExternalUpdateAvailable(false);
  };

  const handleSqlChange = (sql: string) => {
    dispatch(setQuerySql(sql));

    if (activeSourceQuery && sql === activeSourceQuery.sql) {
      setLastLoadedSql(activeSourceQuery.sql);
      setExternalUpdateAvailable(false);
    }
  };

  const handleReloadSource = () => {
    if (!activeSourceQuery) {
      return;
    }

    dispatch(setQuerySql(activeSourceQuery.sql));
    setLastLoadedSql(activeSourceQuery.sql);
    setExternalUpdateAvailable(false);
  };

  const handleExecuteSql = (sql: string) => {
    setExecutionMode("sql");
    executeQuery({ sql });
  };

  const handleExecuteCommand = async () => {
    if (!activeCommand) {
      return;
    }

    setExecutionMode("command");
    try {
      const result = await executeQueryCommand({
        path: activeCommand.path,
        values: activeCommandValues,
      }).unwrap();
      if (result.rendered_sql) {
        dispatch(setQuerySql(result.rendered_sql));
      }
    } catch {
      // RTK Query keeps the structured error state in commandExecution.
    }
  };

  const displayedResult = executionMode === "command"
    ? commandExecution.data ?? null
    : sqlExecution.data ?? null;
  const displayedError = executionMode === "command"
    ? extractQueryError(commandExecution.error)
    : extractQueryError(sqlExecution.error);
  const isLoading = executionMode === "command"
    ? commandExecution.isLoading
    : sqlExecution.isLoading;

  return (
    <QueryEditor
      sql={queryEditorSql}
      activeCommand={activeCommand}
      commandValues={activeCommandValues}
      onSqlChange={handleSqlChange}
      onCommandValueChange={(name, value) => setActiveCommandValues((prev) => ({ ...prev, [name]: value }))}
      onExecute={handleExecuteSql}
      onExecuteCommand={handleExecuteCommand}
      onSave={(s) =>
        saveQuery({
          name: `query-${Date.now()}`,
          folder: "my-queries",
          description: "",
          sql: s,
        })
      }
      onSelectQuery={(query, kind) => handleSelectQuery(kind)(query)}
      onSelectCommand={handleSelectCommand}
      onReloadSource={handleReloadSource}
      onClickSessionId={(id) => navigate(`/sessions/${id}`)}
      presets={presets}
      savedQueries={savedQueries}
      commands={queryCommands}
      result={displayedResult}
      error={displayedError}
      isLoading={isLoading}
      sourceStatus={
        activeSource
          ? {
              label: activeSource.kind === "preset"
                ? "Preset file"
                : activeSource.kind === "saved"
                  ? "Saved query file"
                  : "Query command",
              path: activeSource.path,
              missing: activeSource.kind === "command" ? activeCommand === null : activeSourceQuery === null,
              externalUpdateAvailable: activeSource.kind === "command" ? false : externalUpdateAvailable,
            }
          : null
      }
    />
  );
}

function buildInitialCommandValues(command: QueryCommand): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const param of [...command.arguments, ...command.flags]) {
    const parsed = parseDefaultValue(param);
    if (parsed !== undefined) {
      values[param.name] = parsed;
    }
  }
  return values;
}

function parseDefaultValue(param: QueryCommandParam): unknown {
  if (!param.defaultJson) {
    return undefined;
  }

  try {
    return JSON.parse(param.defaultJson);
  } catch {
    return undefined;
  }
}

function extractQueryError(error: unknown): { message: string } | null {
  if (!error || typeof error !== "object" || !("data" in error)) {
    return null;
  }

  const data = (error as { data?: { error?: { message?: string } | string } }).data;
  if (!data) {
    return null;
  }
  if (typeof data.error === "string") {
    return { message: data.error };
  }
  if (data.error?.message) {
    return { message: data.error.message };
  }
  return null;
}
