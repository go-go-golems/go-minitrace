import type { Meta, StoryObj } from "@storybook/react";
import Box from "@mui/material/Box";
import { useState } from "react";
import { AnnotationModal } from "../AnnotationModal";
import { withTheme } from "../../../test-utils/storybook-decorators";
import Button from "@mui/material/Button";

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

// Wrapper to manage modal state in stories
function ModalWrapper({ target }: { target: { scopeType: "session" | "turn" | "tool_call"; targetId: string } | null }) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open Modal</Button>
      <AnnotationModal
        sessionId="demo-session-123"
        target={open ? target : null}
        onClose={() => setOpen(false)}
      />
    </>
  );
}

export const SessionAnnotation: Story = {
  render: () => (
    <ModalWrapper target={{ scopeType: "session", targetId: "demo-session-123" }} />
  ),
};

export const TurnAnnotation: Story = {
  render: () => (
    <ModalWrapper target={{ scopeType: "turn", targetId: "42" }} />
  ),
};

export const ToolCallAnnotation: Story = {
  render: () => (
    <ModalWrapper target={{ scopeType: "tool_call", targetId: "call_abc123" }} />
  ),
};

export const Closed: Story = {
  render: () => (
    <ModalWrapper target={null} />
  ),
};
