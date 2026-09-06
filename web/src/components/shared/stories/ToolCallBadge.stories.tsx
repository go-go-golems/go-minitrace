import type { Meta, StoryObj } from "@storybook/react-vite";
import Stack from "@mui/material/Stack";
import { ToolCallBadgeChip } from "../ToolCallBadge";
import { withTheme } from "../../../test-utils/storybook-decorators";
import type { ToolCallBadge } from "../../../types";

const meta = {
  title: "Shared/ToolCallBadge",
  component: ToolCallBadgeChip,
  decorators: [withTheme],
  parameters: { layout: "centered" },
} satisfies Meta<typeof ToolCallBadgeChip>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Commit: Story = { args: { badge: "commit" } };
export const TicketCreate: Story = { args: { badge: "ticket-create" } };
export const DocAdd: Story = { args: { badge: "doc-add" } };
export const DiaryWrite: Story = { args: { badge: "diary-write" } };
export const Error: Story = { args: { badge: "error" } };

export const AllBadges: Story = {
  args: { badge: "commit" },
  render: () => (
    <Stack direction="row" spacing={1}>
      {(["commit", "ticket-create", "doc-add", "diary-write", "error"] as ToolCallBadge[]).map(
        (b) => (
          <ToolCallBadgeChip key={b} badge={b} />
        )
      )}
    </Stack>
  ),
};
