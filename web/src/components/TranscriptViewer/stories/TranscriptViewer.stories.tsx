import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "storybook/test";
import Box from "@mui/material/Box";
import { TranscriptViewer } from "../TranscriptViewer";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockSessionDetail } from "../../../mocks/data";

const meta = {
  title: "Screens/TranscriptViewer",
  component: TranscriptViewer,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ height: "100vh", bgcolor: "background.default" }}>
        <Story />
      </Box>
    ),
  ],
  args: {
    session: mockSessionDetail,
    onBack: fn(),
    onQuerySession: fn(),
  },
} satisfies Meta<typeof TranscriptViewer>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const LongTitle: Story = {
  args: {
    session: {
      ...mockSessionDetail,
      title:
        "This is a very long session title that should be truncated properly in the header because it contains a lot of information about what was done during this extremely productive coding session where many things were accomplished including refactoring and testing",
    },
  },
};

export const HighActivitySession: Story = {
  args: {
    session: {
      ...mockSessionDetail,
      timing: {
        ...mockSessionDetail.timing,
        duration_seconds: 4800,
        active_duration_seconds: 3600,
      },
    },
  },
};

export const SingleBlock: Story = {
  args: {
    session: {
      ...mockSessionDetail,
      blocks: [mockSessionDetail.blocks[0]],
    },
  },
};
