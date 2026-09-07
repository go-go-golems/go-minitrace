import type { Meta, StoryObj } from "@storybook/react-vite";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { FormatDuration, FormatWallActive } from "../FormatDuration";
import { withTheme } from "../../../test-utils/storybook-decorators";

const meta = {
  title: "Shared/FormatDuration",
  component: FormatDuration,
  decorators: [withTheme],
  parameters: { layout: "centered" },
} satisfies Meta<typeof FormatDuration>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Seconds: Story = { args: { seconds: 45 } };
export const Minutes: Story = { args: { seconds: 2700 } };
export const Hours: Story = { args: { seconds: 88896 } };

export const AllDurations: Story = {
  args: { seconds: 0 },
  render: () => (
    <Stack spacing={1}>
      {[12, 45, 120, 2700, 7200, 45605, 88896, 315000].map((s) => (
        <Stack key={s} direction="row" spacing={2} alignItems="center">
          <Typography variant="caption" sx={{ width: 80, fontFamily: "monospace", textAlign: "right" }}>
            {s}s
          </Typography>
          <FormatDuration seconds={s} />
        </Stack>
      ))}
    </Stack>
  ),
};

export const WallActive: StoryObj = {
  render: () => (
    <Stack spacing={1}>
      <FormatWallActive wallSeconds={88896} activeSeconds={11160} />
      <FormatWallActive wallSeconds={315000} activeSeconds={44280} />
      <FormatWallActive wallSeconds={4653} activeSeconds={2880} />
    </Stack>
  ),
};
