import type { Meta, StoryObj } from "@storybook/react";
import Box from "@mui/material/Box";
import { BlockCard } from "../BlockCard";
import { withTheme } from "../../../test-utils/storybook-decorators";
import { mockSessionDetail } from "../../../mocks/data";

const meta = {
  title: "TranscriptViewer/BlockCard",
  component: BlockCard,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ p: 2, bgcolor: "background.default", maxWidth: 900 }}>
        <Story />
      </Box>
    ),
  ],
} satisfies Meta<typeof BlockCard>;

export default meta;
type Story = StoryObj<typeof meta>;

const blocks = mockSessionDetail.blocks;

export const FirstBlock: Story = {
  args: { block: blocks[0], defaultExpanded: true },
};

export const QuestionBlock: Story = {
  args: { block: blocks[1], defaultExpanded: true },
};

export const ContinueBlock: Story = {
  args: { block: blocks[2], defaultExpanded: true },
};

export const CollapsedFirst: Story = {
  args: { block: blocks[0], defaultExpanded: false },
};

export const CollapsedContinue: Story = {
  args: { block: blocks[2], defaultExpanded: false },
};

export const WithLargeGap: Story = {
  args: {
    block: {
      ...blocks[1],
      gap_minutes: 609,
    },
    defaultExpanded: false,
  },
};

export const WithSmallGap: Story = {
  args: {
    block: {
      ...blocks[2],
      gap_minutes: 5,
    },
    defaultExpanded: false,
  },
};

export const WithManyArtifacts: Story = {
  args: {
    block: {
      ...blocks[0],
      artifacts: {
        commits: [
          "refactor(profilechat): resolve engine profiles",
          "refactor(pinoweb): align inventory wrappers",
          "feat(profilechat): support configured default",
        ],
        tickets_created: ["APP-30-WESEN-OS-PINOCCHIO-PROFILE-BOOTSTRAP"],
        docs_added: [
          "Investigation diary (reference)",
          "Intern guide to migrating... (design-doc)",
          "Implementation Postmortem (design-doc)",
        ],
        diary_writes: 3,
      },
    },
    defaultExpanded: true,
  },
};

export const NoArtifacts: Story = {
  args: {
    block: {
      ...blocks[1],
      artifacts: { commits: [], tickets_created: [], docs_added: [], diary_writes: 0 },
    },
    defaultExpanded: true,
  },
};

export const AllBlocksStacked: Story = {
  args: { block: blocks[0] },
  render: () => (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
      {blocks.map((b) => (
        <BlockCard key={b.block_num} block={b} defaultExpanded={false} />
      ))}
    </Box>
  ),
};
