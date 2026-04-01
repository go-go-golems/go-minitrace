import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import Alert from "@mui/material/Alert";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
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
  const [savedFlash, setSavedFlash] = useState(false);

  const handleSave = () => {
    onSave?.(sql);
    setSavedFlash(true);
    setTimeout(() => setSavedFlash(false), 2000);
  };

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

      {/* Main pane — vertical split: editor top, results bottom */}
      <Box sx={{ flex: 1, display: "flex", flexDirection: "column", overflow: "hidden" }}>
        {/* Editor section */}
        <Box
          sx={{
            flex: "0 0 auto",
            minHeight: 160,
            maxHeight: "40vh",
            display: "flex",
            flexDirection: "column",
            p: 2,
            pb: 1,
          }}
        >
          <Box sx={{ flex: 1, minHeight: 0 }}>
            <SqlEditor
              value={sql}
              onChange={onSqlChange}
              onExecute={() => onExecute(sql)}
            />
          </Box>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 1 }}>
            <Button
              variant="contained"
              startIcon={
                isLoading ? <CircularProgress size={14} color="inherit" /> : <PlayArrowIcon />
              }
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
                onClick={handleSave}
                size="small"
              >
                Save
              </Button>
            )}
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ ml: 1 }}
            >
              Ctrl+Enter to run
            </Typography>
            {savedFlash && (
              <Typography variant="caption" color="success.main">
                ✓ Saved
              </Typography>
            )}
          </Stack>
        </Box>

        <Divider />

        {/* Results section */}
        <Box sx={{ flex: 1, overflow: "auto", px: 2, py: 1 }}>
          {error && (
            <Alert
              severity="error"
              sx={{
                mb: 1,
                fontFamily: "monospace",
                fontSize: "0.8rem",
                "& .MuiAlert-message": { whiteSpace: "pre-wrap", wordBreak: "break-all" },
              }}
            >
              {error.message}
            </Alert>
          )}
          {result && (
            <ResultsTable result={result} onClickSessionId={onClickSessionId} />
          )}
          {!result && !error && !isLoading && (
            <Box
              sx={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                height: "100%",
                opacity: 0.3,
              }}
            >
              <Typography variant="body1">
                Run a query to see results
              </Typography>
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  );
}
