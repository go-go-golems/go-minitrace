import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import Box from "@mui/material/Box";
import { QuerySidebar } from "../QuerySidebar";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockPresets, mockSavedQueries } from "../../../mocks/data";

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
    onSelect: fn<(query: unknown, kind: "preset" | "saved") => void>(),
  },
} satisfies Meta<typeof QuerySidebar>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const EmptyPresets: Story = {
  args: { presets: [] },
};

export const EmptySaved: Story = {
  args: { savedQueries: [] },
};

export const ManyQueries: Story = {
  args: {
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
