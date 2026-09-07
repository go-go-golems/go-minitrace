import type { Meta, StoryObj } from "@storybook/react-vite";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { ActiveBadge } from "../ActiveBadge";
import { withTheme } from "../../../test-utils/storybook-decorators";

const meta = {
  title: "Shared/ActiveBadge",
  component: ActiveBadge,
  decorators: [withTheme],
  parameters: { layout: "centered" },
} satisfies Meta<typeof ActiveBadge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const HighActivity: Story = { args: { activePct: 75 } };
export const MediumActivity: Story = { args: { activePct: 25 } };
export const LowActivity: Story = { args: { activePct: 5 } };
export const Boundary50: Story = { args: { activePct: 50 } };
export const Boundary10: Story = { args: { activePct: 10 } };

export const AllVariants: Story = {
  args: { activePct: 0 },
  render: () => (
    <Stack spacing={2} alignItems="center">
      {[85, 61, 50, 35, 14, 10, 7, 2].map((pct) => (
        <Stack key={pct} direction="row" spacing={2} alignItems="center">
          <Typography variant="body2" sx={{ width: 40, textAlign: "right", fontFamily: "monospace" }}>
            {pct}%
          </Typography>
          <ActiveBadge activePct={pct} />
        </Stack>
      ))}
    </Stack>
  ),
};
