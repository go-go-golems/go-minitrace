import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";

export type ActiveView = "sessions" | "transcript" | "query";

interface UiState {
  activeView: ActiveView;
  selectedSessionId: string | null;
  queryEditorSql: string;
  filterText: string;
}

const initialState: UiState = {
  activeView: "sessions",
  selectedSessionId: null,
  queryEditorSql: "SELECT session_id, title, turn_count AS turns\nFROM sessions\nORDER BY started_at\nLIMIT 20;",
  filterText: "",
};

function escapeSQLLiteral(value: string): string {
  return value.replaceAll("'", "''");
}

const uiSlice = createSlice({
  name: "ui",
  initialState,
  reducers: {
    setActiveView(state, action: PayloadAction<ActiveView>) {
      state.activeView = action.payload;
    },
    selectSession(state, action: PayloadAction<string>) {
      state.selectedSessionId = action.payload;
      state.activeView = "transcript";
    },
    openQueryForSession(state, action: PayloadAction<string>) {
      const escapedSessionID = escapeSQLLiteral(action.payload);
      state.queryEditorSql = `-- Session: ${action.payload}\nSELECT tool_call_id, tool_name, operation_type,\n  success, substr(COALESCE(command, file_path, ''), 1, 200) AS target\nFROM tool_calls\nWHERE session_id = '${escapedSessionID}'\nORDER BY timestamp\nLIMIT 50;`;
      state.activeView = "query";
    },
    setQuerySql(state, action: PayloadAction<string>) {
      state.queryEditorSql = action.payload;
    },
    loadPresetQuery(state, action: PayloadAction<string>) {
      state.queryEditorSql = action.payload;
      state.activeView = "query";
    },
    setFilterText(state, action: PayloadAction<string>) {
      state.filterText = action.payload;
    },
  },
});

export const {
  setActiveView,
  selectSession,
  openQueryForSession,
  setQuerySql,
  loadPresetQuery,
  setFilterText,
} = uiSlice.actions;

export const uiReducer = uiSlice.reducer;
