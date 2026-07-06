import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";
import Box from "@mui/material/Box";
import { QueryCommandForm } from "../QueryCommandForm";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockQueryCommands } from "../../../mocks/data";
import type { QueryCommand } from "../../../types";

const meta = {
  title: "QueryEditor/QueryCommandForm",
  component: QueryCommandForm,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ width: 520, p: 2, bgcolor: "background.default" }}>
        <Story />
      </Box>
    ),
  ],
  args: {
    command: mockQueryCommands[0],
    values: {
      framework: ["codex"],
      title_like: "wesen",
      limit: 50,
    },
    renderedSql: "SELECT session_id AS id, title FROM sessions WHERE LOWER(title) LIKE LOWER('%wesen%') LIMIT 50;",
    onChange: fn(),
  },
} satisfies Meta<typeof QueryCommandForm>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const AliasDefaults: Story = {
  args: {
    command: mockQueryCommands[1],
    values: {
      framework: ["codex"],
    },
  },
};

const advancedFieldCommand: QueryCommand = {
  name: "advanced-fields",
  folder: "analysis",
  path: "analysis/advanced-fields.sql",
  shortDescription: "Exercise float/date/floatList/choiceList field rendering.",
  longDescription: "Story-only command used to verify richer field widgets.",
  flags: [
    {
      name: "threshold",
      type: "float",
      help: "Minimum score threshold",
      required: false,
      defaultJson: "0.75",
      choices: [],
      positional: false,
      shortFlag: "",
    },
    {
      name: "window_start",
      type: "date",
      help: "Only include rows on or after this date",
      required: false,
      defaultJson: '"2026-04-01"',
      choices: [],
      positional: false,
      shortFlag: "",
    },
    {
      name: "bins",
      type: "floatList",
      help: "Float bucket edges",
      required: false,
      defaultJson: "[0.1,0.5,0.9]",
      choices: [],
      positional: false,
      shortFlag: "",
    },
    {
      name: "frameworks",
      type: "choiceList",
      help: "Frameworks to include",
      required: false,
      defaultJson: '["codex","pi"]',
      choices: ["codex", "pi", "claude"],
      positional: false,
      shortFlag: "",
    },
  ],
  arguments: [],
  tags: ["story"],
  readonly: true,
  kind: "verb",
  aliasFor: "",
  rawSqlPath: "analysis/advanced-fields.sql",
  rawSql: "SELECT 1;",
};

export const AdvancedFieldTypes: Story = {
  args: {
    command: advancedFieldCommand,
    values: {
      threshold: 0.8,
      window_start: "2026-04-03",
      bins: [0.2, 0.4, 0.6],
      frameworks: ["codex", "pi"],
    },
    renderedSql: "SELECT 1;",
  },
};
