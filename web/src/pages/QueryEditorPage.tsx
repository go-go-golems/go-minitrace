import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useSelector, useDispatch } from "react-redux";
import {
  useExecuteQueryMutation,
  useGetPresetsQuery,
  useGetSavedQueriesQuery,
  useSaveQueryMutation,
} from "../api/minitrace";
import { QueryEditor } from "../components/QueryEditor";
import type { RootState, AppDispatch } from "../store";
import { setQuerySql, openQueryForSession } from "../store";

export function QueryEditorPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const dispatch = useDispatch<AppDispatch>();
  const { queryEditorSql } = useSelector((state: RootState) => state.ui);

  const { data: presets = [] } = useGetPresetsQuery();
  const { data: savedQueries = [] } = useGetSavedQueriesQuery();
  const [executeQuery, { data: queryResult, error: queryError, isLoading }] =
    useExecuteQueryMutation();
  const [saveQuery] = useSaveQueryMutation();

  // If ?session=xxx is in the URL, populate the editor
  useEffect(() => {
    const sessionId = searchParams.get("session");
    if (sessionId) {
      dispatch(openQueryForSession(sessionId));
    }
  }, [searchParams, dispatch]);

  return (
    <QueryEditor
      sql={queryEditorSql}
      onSqlChange={(s) => dispatch(setQuerySql(s))}
      onExecute={(s) => executeQuery({ sql: s })}
      onSave={(s) =>
        saveQuery({
          name: `query-${Date.now()}`,
          folder: "my-queries",
          description: "",
          sql: s,
        })
      }
      onSelectPreset={(s) => dispatch(setQuerySql(s))}
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
    />
  );
}
