import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import Alert from "@mui/material/Alert";
import CircularProgress from "@mui/material/CircularProgress";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import SaveIcon from "@mui/icons-material/Save";
import type { SavedQuery, QueryResult, QueryError } from "../../types";
import { SqlEditor } from "./SqlEditor";
import { QuerySidebar } from "./QuerySidebar";
import { ResultsTable } from "./ResultsTable";

interface QueryEditorProps {
  sql: string;
  onSqlChange: (sql: string) => void;
  onExecute: (sql: string) => void;
  onSave?: (sql: string) => void;
  onSelectPreset: (sql: string) => void;
  onClickSessionId?: (id: string) => void;
  presets: SavedQuery[];
  savedQueries: SavedQuery[];
  result: QueryResult | null;
  error: QueryError | null;
  isLoading: boolean;
}

export function QueryEditor({
  sql,
  onSqlChange,
  onExecute,
  onSave,
  onSelectPreset,
  onClickSessionId,
  presets,
  savedQueries,
  result,
  error,
  isLoading,
}: QueryEditorProps) {
  const [showSave, setShowSave] = useState(false);

  return (
    <Box
      data-widget="query-editor"
      sx={{ height: "100%", display: "flex" }}
    >
      {/* Sidebar */}
      <QuerySidebar
        presets={presets}
        savedQueries={savedQueries}
        onSelect={onSelectPreset}
      />

      {/* Main pane */}
      <Box sx={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
        {/* Editor */}
        <Box sx={{ flex: "0 0 auto", height: 220, p: 2, display: "flex", flexDirection: "column" }}>
          <Box sx={{ flex: 1, minHeight: 0 }}>
            <SqlEditor
              value={sql}
              onChange={onSqlChange}
              onExecute={() => onExecute(sql)}
            />
          </Box>
          <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
            <Button
              variant="contained"
              startIcon={isLoading ? <CircularProgress size={16} /> : <PlayArrowIcon />}
              onClick={() => onExecute(sql)}
              disabled={isLoading}
              size="small"
            >
              Run
            </Button>
            {onSave && (
              <Button
                variant="outlined"
                startIcon={<SaveIcon />}
                onClick={() => {
                  setShowSave(true);
                  onSave(sql);
                }}
                size="small"
              >
                Save
              </Button>
            )}
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ alignSelf: "center", ml: 2 }}
            >
              Ctrl+Enter to run
            </Typography>
            {showSave && (
              <Typography
                variant="caption"
                color="success.main"
                sx={{ alignSelf: "center" }}
              >
                ✓ Saved
              </Typography>
            )}
          </Stack>
        </Box>

        {/* Results */}
        <Box sx={{ flex: 1, overflow: "auto", px: 2, pb: 2 }}>
          {error && (
            <Alert severity="error" sx={{ mb: 1, fontFamily: "monospace", fontSize: "0.8rem" }}>
              {error.message}
            </Alert>
          )}
          {result && (
            <ResultsTable result={result} onClickSessionId={onClickSessionId} />
          )}
          {!result && !error && !isLoading && (
            <Box sx={{ display: "flex", alignItems: "center", justifyContent: "center", height: 200, opacity: 0.4 }}>
              <Typography variant="body2">
                Run a query to see results
              </Typography>
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  );
}
