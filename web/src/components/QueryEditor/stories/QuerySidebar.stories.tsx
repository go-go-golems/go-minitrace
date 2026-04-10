import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";
import Box from "@mui/material/Box";
import { QuerySidebar } from "../QuerySidebar";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockPresets, mockQueryCommands, mockSavedQueries } from "../../../mocks/data";

const meta = {
  title: "QueryEditor/QuerySidebar",
  component: QuerySidebar,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ height: 600, bgcolor: "background.default", display: "flex" }}>
        <Story />
      </Box>
    ),
  ],
  args: {
    presets: mockPresets,
    savedQueries: mockSavedQueries,
    commands: mockQueryCommands,
    onSelect: fn<(query: unknown, kind: "preset" | "saved") => void>(),
    onSelectCommand: fn<(command: unknown) => void>(),
  },
} satisfies Meta<typeof QuerySidebar>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const EmptyCommands: Story = {
  args: { commands: [] },
};

export const EmptyPresets: Story = {
  args: { presets: [] },
};

export const EmptySaved: Story = {
  args: { savedQueries: [] },
};

export const ManyQueries: Story = {
  args: {
    commands: [
      ...mockQueryCommands,
      ...Array.from({ length: 6 }, (_, i) => ({
        name: `command-${i + 3}`,
        folder: i < 3 ? "core" : "analysis",
        path: `${i < 3 ? "core" : "analysis"}/command-${i + 3}.sql`,
        shortDescription: `Generated command ${i + 3}`,
        longDescription: "",
        flags: [],
        arguments: [],
        tags: [],
        readonly: true,
        kind: "verb" as const,
        aliasFor: "",
        rawSqlPath: `${i < 3 ? "core" : "analysis"}/command-${i + 3}.sql`,
        rawSql: `SELECT ${i + 3} AS generated_command_${i + 3};`,
      })),
    ],
    presets: [
      ...mockPresets,
      ...Array.from({ length: 10 }, (_, i) => ({
        name: `preset-${i + 5}`,
        folder: i < 5 ? "core" : "advanced",
        path: `${i < 5 ? "core" : "advanced"}/preset-${i + 5}.sql`,
        description: `Generated preset number ${i + 5}`,
        sql: `SELECT ${i + 5} AS num;`,
        readonly: true,
      })),
    ],
    savedQueries: [
      ...mockSavedQueries,
      ...Array.from({ length: 6 }, (_, i) => ({
        name: `my-query-${i + 2}`,
        folder: "my-queries",
        path: `my-queries/my-query-${i + 2}.sql`,
        description: `Custom query ${i + 2}`,
        sql: `SELECT '${i + 2}';`,
        readonly: false,
      })),
    ],
  },
};
