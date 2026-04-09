import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";
import Box from "@mui/material/Box";
import { QueryCommandForm } from "../QueryCommandForm";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockQueryCommands } from "../../../mocks/data";

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
