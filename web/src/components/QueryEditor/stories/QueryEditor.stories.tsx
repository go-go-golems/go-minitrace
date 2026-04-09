import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";
import Box from "@mui/material/Box";
import { QueryEditor } from "../QueryEditor";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockPresets, mockQueryCommands, mockSavedQueries, mockQueryResult } from "../../../mocks/data";

const meta = {
  title: "Screens/QueryEditor",
  component: QueryEditor,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ height: "100vh", bgcolor: "background.default" }}>
        <Story />
      </Box>
    ),
  ],
  args: {
    sql: "SELECT id, title,\n  CAST(metrics->>'turn_count' AS INT) AS turns\nFROM sessions_base\nWHERE LOWER(title) LIKE '%wesen%'\nORDER BY timing->>'started_at';",
    onSqlChange: fn(),
    onExecute: fn(),
    onExecuteCommand: fn(),
    onSave: fn(),
    onSelectQuery: fn(),
    onSelectCommand: fn(),
    onCommandValueChange: fn(),
    onReloadSource: fn(),
    onClickSessionId: fn(),
    presets: mockPresets,
    savedQueries: mockSavedQueries,
    commands: mockQueryCommands,
    result: null,
    error: null,
    isLoading: false,
    sourceStatus: {
      label: "Preset file",
      path: "core/sessions-for-wesen.sql",
      missing: false,
      externalUpdateAvailable: false,
    },
  },
} satisfies Meta<typeof QueryEditor>;

export default meta;
type Story = StoryObj<typeof meta>;

export const EmptyState: Story = {};

export const WithResults: Story = {
  args: {
    result: mockQueryResult,
  },
};

export const WithError: Story = {
  args: {
    sql: "SELECT error_column FROM sessions_base;",
    error: {
      message: 'Binder Error: Referenced column "error_column" not found in FROM clause!\nCandidate bindings: "title", "summary", "operational_context"',
    },
  },
};

export const Loading: Story = {
  args: {
    isLoading: true,
  },
};

export const LongQuery: Story = {
  args: {
    sql: `-- Extract all user turns with timestamps and gaps
WITH user_turns AS (
  SELECT
    s.id AS session_id,
    t.idx AS turn_idx,
    CAST(t.turn->>'timestamp' AS TIMESTAMP) AS ts,
    LEFT(CAST(t.turn->>'content' AS VARCHAR), 200) AS content
  FROM sessions_base s
  CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)
  WHERE s.id = '019d174c-fc68-7c00-8f1b-7fcc067c1fd6'
    AND CAST(t.turn->>'role' AS VARCHAR) = 'user'
),
gaps AS (
  SELECT *,
    ts - LAG(ts) OVER (PARTITION BY session_id ORDER BY turn_idx) AS gap
  FROM user_turns
)
SELECT session_id, turn_idx, ts,
  ROUND(EXTRACT(EPOCH FROM gap) / 60, 1) AS gap_minutes,
  content
FROM gaps
WHERE EXTRACT(EPOCH FROM gap) > 1800
ORDER BY ts;`,
    result: {
      columns: ["session_id", "turn_idx", "ts", "gap_minutes", "content"],
      rows: [
        { session_id: "019d174c-fc68-7c00-8f1b-7fcc067c1fd6", turn_idx: 25, ts: "2026-03-23T02:36:27", gap_minutes: 349, content: "how does this all impact go-go-os-chat?" },
        { session_id: "019d174c-fc68-7c00-8f1b-7fcc067c1fd6", turn_idx: 121, ts: "2026-03-23T13:59:32", gap_minutes: 609, content: "Ok, we used to have an analyst profile before..." },
      ],
      duration_ms: 45,
      row_count: 2,
    },
  },
};

export const CommandMode: Story = {
  args: {
    activeCommand: mockQueryCommands[0],
    commandValues: {
      framework: ["codex"],
      title_like: "wesen",
      limit: 25,
    },
    sourceStatus: {
      label: "Query command",
      path: mockQueryCommands[0].path,
      missing: false,
      externalUpdateAvailable: false,
    },
    result: mockQueryResult,
    onSave: undefined,
  },
};

export const NoSaveButton: Story = {
  args: {
    onSave: undefined,
    result: mockQueryResult,
  },
};

export const ReloadAvailable: Story = {
  args: {
    sourceStatus: {
      label: "Saved query file",
      path: "my-queries/scratchpad.sql",
      missing: false,
      externalUpdateAvailable: true,
    },
  },
};

export const MissingSource: Story = {
  args: {
    sourceStatus: {
      label: "Saved query file",
      path: "my-queries/deleted.sql",
      missing: true,
      externalUpdateAvailable: false,
    },
  },
};
