import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import Box from "@mui/material/Box";
import { SessionBrowser } from "../SessionBrowser";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockSessions } from "../../../mocks/data";

const meta = {
  title: "Screens/SessionBrowser",
  component: SessionBrowser,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ height: "100vh", bgcolor: "background.default" }}>
        <Story />
      </Box>
    ),
  ],
  args: {
    sessions: mockSessions,
    filterText: "",
    onFilterChange: fn(),
    onSelectSession: fn(),
    onQuerySession: fn(),
  },
} satisfies Meta<typeof SessionBrowser>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithFilter: Story = {
  args: { filterText: "wesen" },
};

export const FilterByModel: Story = {
  args: { filterText: "gpt-5.4" },
};

export const SingleResult: Story = {
  args: { filterText: "019d174c" },
};

export const NoResults: Story = {
  args: { filterText: "nonexistent-query-that-matches-nothing" },
};

export const EmptySessions: Story = {
  args: { sessions: [] },
};

export const ManySessions: Story = {
  args: {
    sessions: Array.from({ length: 50 }, (_, i) => ({
      ...mockSessions[i % mockSessions.length],
      id: `session-${i}-${Math.random().toString(36).slice(2, 10)}`,
      title: `Session ${i + 1}: ${mockSessions[i % mockSessions.length].title}`,
      timing: {
        ...mockSessions[i % mockSessions.length].timing,
        duration_seconds: Math.random() * 200000,
        active_duration_seconds: Math.random() * 50000,
      },
    })),
  },
};
