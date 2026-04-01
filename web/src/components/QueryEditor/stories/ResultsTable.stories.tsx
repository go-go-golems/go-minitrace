import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import Box from "@mui/material/Box";
import { ResultsTable } from "../ResultsTable";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockQueryResult, mockSessions } from "../../../mocks/data";

const meta = {
  title: "QueryEditor/ResultsTable",
  component: ResultsTable,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ p: 2, bgcolor: "background.default", maxWidth: 1000 }}>
        <Story />
      </Box>
    ),
  ],
  args: {
    result: mockQueryResult,
    onClickSessionId: fn(),
  },
} satisfies Meta<typeof ResultsTable>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const SingleRow: Story = {
  args: {
    result: {
      columns: ["id", "title", "turns"],
      rows: [{ id: mockSessions[0].id, title: mockSessions[0].title, turns: 315 }],
      duration_ms: 3,
      row_count: 1,
    },
  },
};

export const ManyColumns: Story = {
  args: {
    result: {
      columns: ["id", "started_at", "title", "turns", "tools", "model", "workdir", "hours"],
      rows: mockSessions.map((s) => ({
        id: s.id,
        started_at: s.timing.started_at,
        title: s.title,
        turns: s.metrics.turn_count,
        tools: s.metrics.tool_call_count,
        model: s.environment.model,
        workdir: s.operational_context.working_directory,
        hours: (s.timing.duration_seconds / 3600).toFixed(1),
      })),
      duration_ms: 8,
      row_count: mockSessions.length,
    },
  },
};

export const LongTextValues: Story = {
  args: {
    result: {
      columns: ["turn", "role", "content"],
      rows: [
        { turn: 1, role: "user", content: "Read the docs in geppetto and create a new ticket to bring wesen-os/ up to the new profiles and profile registry settings, and have it load the pinocchio config and profile files (not wesen-os ones). Create the docmgr ticket, keep a diary. Then create a detailed analysis / design / implementation guide..." },
        { turn: 2, role: "assistant", content: "Using ticket-research-docmgr-remarkable with diary support because this is a ticketed research/doc deliverable that needs a structured workspace, diary, and reMarkable upload. I'm starting by reading the skill instructions and locating the relevant docs..." },
      ],
      duration_ms: 5,
      row_count: 2,
    },
  },
};

export const EmptyResult: Story = {
  args: {
    result: {
      columns: ["id", "title"],
      rows: [],
      duration_ms: 2,
      row_count: 0,
    },
  },
};

export const NumericHeavy: Story = {
  args: {
    result: {
      columns: ["tool", "calls", "avg_duration_ms"],
      rows: [
        { tool: "exec_command", calls: 2578, avg_duration_ms: 1250 },
        { tool: "write_stdin", calls: 664, avg_duration_ms: 150 },
        { tool: "update_plan", calls: 31, avg_duration_ms: 2000 },
        { tool: "mcp__playwright__browser_navigate", calls: 6, avg_duration_ms: 3500 },
        { tool: "mcp__codex_apps__github_create_pull_request", calls: 3, avg_duration_ms: 8000 },
      ],
      duration_ms: 15,
      row_count: 5,
    },
  },
};
