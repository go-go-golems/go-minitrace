import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useSelector, useDispatch } from "react-redux";
import {
  useExecuteQueryMutation,
  useGetPresetsQuery,
  useGetSavedQueriesQuery,
  useSaveQueryMutation,
} from "../api/minitrace";
import { QueryEditor } from "../components/QueryEditor";
import type { SavedQuery } from "../types";
import type { RootState, AppDispatch } from "../store";
import { setQuerySql, openQueryForSession } from "../store";

type QuerySourceKind = "preset" | "saved";

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
  const [lastLoadedSql, setLastLoadedSql] = useState<string | null>(null);
  const [externalUpdateAvailable, setExternalUpdateAvailable] = useState(false);

  const { data: presets = [] } = useGetPresetsQuery(undefined, {
    pollingInterval: 3000,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });
  const { data: savedQueries = [] } = useGetSavedQueriesQuery(undefined, {
    pollingInterval: 3000,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });
  const [executeQuery, { data: queryResult, error: queryError, isLoading }] =
    useExecuteQueryMutation();
  const [saveQuery] = useSaveQueryMutation();

  const activeSourceQuery = useMemo(() => {
    if (!activeSource) {
      return null;
    }

    const queries = activeSource.kind === "preset" ? presets : savedQueries;
    return queries.find((query) => query.path === activeSource.path) ?? null;
  }, [activeSource, presets, savedQueries]);

  // If ?session=xxx is in the URL, populate the editor
  useEffect(() => {
    const sessionId = searchParams.get("session");
    if (sessionId) {
      setActiveSource(null);
      setLastLoadedSql(null);
      setExternalUpdateAvailable(false);
      dispatch(openQueryForSession(sessionId));
    }
  }, [searchParams, dispatch]);

  useEffect(() => {
    if (!activeSource || !activeSourceQuery) {
      return;
    }

    const latestSql = activeSourceQuery.sql;
    if (lastLoadedSql === null) {
      setLastLoadedSql(latestSql);
      return;
    }

    if (latestSql === lastLoadedSql) {
      if (queryEditorSql === latestSql && externalUpdateAvailable) {
        setExternalUpdateAvailable(false);
      }
      return;
    }

    if (queryEditorSql === lastLoadedSql) {
      dispatch(setQuerySql(latestSql));
      setLastLoadedSql(latestSql);
      setExternalUpdateAvailable(false);
      return;
    }

    setExternalUpdateAvailable(true);
  }, [
    activeSource,
    activeSourceQuery,
    dispatch,
    externalUpdateAvailable,
    lastLoadedSql,
    queryEditorSql,
  ]);

  const handleSelectQuery = (kind: QuerySourceKind) => (query: SavedQuery) => {
    setActiveSource({ kind, path: query.path });
    setLastLoadedSql(query.sql);
    setExternalUpdateAvailable(false);
    dispatch(setQuerySql(query.sql));
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

  return (
    <QueryEditor
      sql={queryEditorSql}
      onSqlChange={handleSqlChange}
      onExecute={(s) => executeQuery({ sql: s })}
      onSave={(s) =>
        saveQuery({
          name: `query-${Date.now()}`,
          folder: "my-queries",
          description: "",
          sql: s,
        })
      }
      onSelectQuery={(query, kind) => handleSelectQuery(kind)(query)}
      onReloadSource={handleReloadSource}
      onClickSessionId={(id) => navigate(`/sessions/${id}`)}
      presets={presets}
      savedQueries={savedQueries}
      result={queryResult ?? null}
      error={
        queryError && "data" in queryError
          ? ((queryError.data as { error?: { message: string } })?.error ?? null)
          : null
      }
      isLoading={isLoading}
      sourceStatus={
        activeSource
          ? {
              label: activeSource.kind === "preset" ? "Preset file" : "Saved query file",
              path: activeSource.path,
              missing: activeSourceQuery === null,
              externalUpdateAvailable,
            }
          : null
      }
    />
  );
}
