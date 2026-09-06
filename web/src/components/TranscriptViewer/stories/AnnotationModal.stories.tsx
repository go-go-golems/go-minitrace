import type { Meta, StoryObj } from "@storybook/react-vite";
import Box from "@mui/material/Box";
import { AnnotationModal } from "../AnnotationModal";
import { withTheme } from "../../../test-utils/storybook-decorators";
import type { AnnotationDraftTarget } from "../AnnotationComposer";

const meta = {
  title: "TranscriptViewer/AnnotationModal",
  component: AnnotationModal,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ p: 2, bgcolor: "background.paper" }}>
        <Story />
      </Box>
    ),
  ],
} satisfies Meta<typeof AnnotationModal>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SessionAnnotation: Story = {
  args: {
    sessionId: "demo-session-123",
    target: { scopeType: "session", targetId: "demo-session-123" } as AnnotationDraftTarget,
    onClose: () => {},
  },
};

export const TurnAnnotation: Story = {
  args: {
    sessionId: "demo-session-123",
    target: { scopeType: "turn", targetId: "42" } as AnnotationDraftTarget,
    onClose: () => {},
  },
};

export const ToolCallAnnotation: Story = {
  args: {
    sessionId: "demo-session-123",
    target: { scopeType: "tool_call", targetId: "call_abc123" } as AnnotationDraftTarget,
    onClose: () => {},
  },
};

export const Closed: Story = {
  args: {
    sessionId: "demo-session-123",
    target: null,
    onClose: () => {},
  },
};
