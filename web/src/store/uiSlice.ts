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
  queryEditorSql: "SELECT id, title,\n  CAST(metrics->>'turn_count' AS INT) AS turns\nFROM sessions_base\nLIMIT 20;",
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
      state.queryEditorSql = `-- Session: ${action.payload}\nSELECT t.idx, CAST(t.turn->>'role' AS VARCHAR) AS role,\n  LEFT(CAST(t.turn->>'content' AS VARCHAR), 500) AS content\nFROM sessions_base\nCROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)\nWHERE id = '${escapedSessionID}'\nORDER BY t.idx\nLIMIT 50;`;
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
