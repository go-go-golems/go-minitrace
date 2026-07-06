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
    sql: "SELECT session_id AS id, title, turn_count AS turns\nFROM sessions\nWHERE LOWER(title) LIKE '%wesen%'\nORDER BY started_at;",
    onSqlChange: fn(),
    onExecute: fn(),
    onExecuteCommand: fn(),
    onPreviewCommand: fn(),
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
    sql: "SELECT error_column FROM sessions;",
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
    session_id,
    turn_index AS turn_idx,
    timestamp AS ts,
    substr(content, 1, 200) AS content
  FROM turns
  WHERE session_id = '019d174c-fc68-7c00-8f1b-7fcc067c1fd6'
    AND role = 'user'
),
gaps AS (
  SELECT *,
    unixepoch(ts) - unixepoch(LAG(ts) OVER (PARTITION BY session_id ORDER BY turn_idx)) AS gap_seconds
  FROM user_turns
)
SELECT session_id, turn_idx, ts,
  ROUND(gap_seconds / 60.0, 1) AS gap_minutes,
  content
FROM gaps
WHERE gap_seconds > 1800
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
    commandRenderedSql: "SELECT session_id AS id, title FROM sessions LIMIT 25;",
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
