import { useSelector, useDispatch } from "react-redux";
import Box from "@mui/material/Box";
import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import DataObjectIcon from "@mui/icons-material/DataObject";
import StorageIcon from "@mui/icons-material/Storage";
import QueryStatsIcon from "@mui/icons-material/QueryStats";
import type { RootState, AppDispatch } from "./store";
import {
  setActiveView,
  selectSession,
  openQueryForSession,
  setQuerySql,
  loadPresetQuery,
  setFilterText,
} from "./store";
import {
  useGetSessionsQuery,
  useGetSessionQuery,
  useExecuteQueryMutation,
  useGetPresetsQuery,
  useGetSavedQueriesQuery,
} from "./api/minitrace";
import { SessionBrowser } from "./components/SessionBrowser";
import { TranscriptViewer } from "./components/TranscriptViewer";
import { QueryEditor } from "./components/QueryEditor";

export function App() {
  const dispatch = useDispatch<AppDispatch>();
  const { activeView, selectedSessionId, queryEditorSql, filterText } =
    useSelector((state: RootState) => state.ui);

  const { data: sessions = [] } = useGetSessionsQuery();
  const { data: sessionDetail } = useGetSessionQuery(selectedSessionId ?? "", {
    skip: !selectedSessionId,
  });
  const { data: presets = [] } = useGetPresetsQuery();
  const { data: savedQueries = [] } = useGetSavedQueriesQuery();
  const [executeQuery, { data: queryResult, error: queryError, isLoading }] =
    useExecuteQueryMutation();

  return (
    <Box sx={{ height: "100vh", display: "flex", flexDirection: "column" }}>
      {/* App bar */}
      <AppBar
        position="static"
        elevation={0}
        sx={{ bgcolor: "background.paper", borderBottom: "1px solid", borderColor: "divider" }}
      >
        <Toolbar variant="dense" sx={{ gap: 1 }}>
          <DataObjectIcon sx={{ color: "primary.main" }} />
          <Typography variant="h4" sx={{ mr: 3, color: "text.primary" }}>
            minitrace
          </Typography>

          <Button
            startIcon={<StorageIcon />}
            onClick={() => dispatch(setActiveView("sessions"))}
            variant={activeView === "sessions" ? "contained" : "text"}
            size="small"
          >
            Sessions
          </Button>
          <Button
            startIcon={<QueryStatsIcon />}
            onClick={() => dispatch(setActiveView("query"))}
            variant={activeView === "query" ? "contained" : "text"}
            size="small"
          >
            Query
          </Button>

          <Box sx={{ flex: 1 }} />
          <Chip
            label={`${sessions.length} sessions loaded`}
            size="small"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
        </Toolbar>
      </AppBar>

      {/* Content */}
      <Box sx={{ flex: 1, overflow: "hidden" }}>
        {activeView === "sessions" && (
          <SessionBrowser
            sessions={sessions}
            filterText={filterText}
            onFilterChange={(t) => dispatch(setFilterText(t))}
            onSelectSession={(id) => dispatch(selectSession(id))}
            onQuerySession={(id) => dispatch(openQueryForSession(id))}
          />
        )}

        {activeView === "transcript" && sessionDetail && (
          <TranscriptViewer
            session={sessionDetail}
            onBack={() => dispatch(setActiveView("sessions"))}
            onQuerySession={(id) => dispatch(openQueryForSession(id))}
          />
        )}

        {activeView === "query" && (
          <QueryEditor
            sql={queryEditorSql}
            onSqlChange={(s) => dispatch(setQuerySql(s))}
            onExecute={(s) => executeQuery({ sql: s })}
            onSelectPreset={(s) => dispatch(loadPresetQuery(s))}
            onClickSessionId={(id) => dispatch(selectSession(id))}
            presets={presets}
            savedQueries={savedQueries}
            result={queryResult ?? null}
            error={
              queryError && "data" in queryError
                ? (queryError.data as { error?: { message: string } })?.error ?? null
                : null
            }
            isLoading={isLoading}
          />
        )}
      </Box>
    </Box>
  );
}
