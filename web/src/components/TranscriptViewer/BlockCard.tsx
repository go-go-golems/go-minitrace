import { memo, useState } from "react";
import Box from "@mui/material/Box";
import Collapse from "@mui/material/Collapse";
import Paper from "@mui/material/Paper";
import type { Annotation, SessionBlock } from "../../types";
import { BlockBody } from "./BlockBody";
import { BlockHeader } from "./BlockHeader";
import type { FocusedTranscriptTarget } from "./types";

interface BlockCardProps {
  block: SessionBlock;
  defaultExpanded?: boolean;
  expanded?: boolean;
  forceExpanded?: boolean;
  focusedTarget?: FocusedTranscriptTarget | null;
  turnAnnotations?: Record<string, Annotation[]>;
  toolCallAnnotations?: Record<string, Annotation[]>;
  onCreateScopedAnnotation?: (
    scopeType: "session" | "turn" | "tool_call",
    targetId: string,
  ) => void;
  onOpenAnnotation?: (annotation: Annotation) => void;
  onToggleExpanded?: () => void;
}

function BlockCardImpl({
  block,
  defaultExpanded = false,
  expanded = false,
  forceExpanded = false,
  focusedTarget = null,
  turnAnnotations = {},
  toolCallAnnotations = {},
  onCreateScopedAnnotation,
  onOpenAnnotation,
  onToggleExpanded,
}: BlockCardProps) {
  const [internalExpanded, setInternalExpanded] = useState(defaultExpanded);
  const [showAllTools, setShowAllTools] = useState(false);
  const isControlled = onToggleExpanded != null;
  const baseExpanded = isControlled ? expanded : expanded || internalExpanded;
  const isExpanded = baseExpanded || forceExpanded;

  const hasFocusedTool = focusedTarget?.scopeType === "tool_call" &&
    block.turns.some((turn) =>
      turn.tool_calls_in_turn.some((tool) => tool.id === focusedTarget.targetId),
    );
  // Keep expansion sticky after focusing a tool, without an effect that causes
  // an additional post-commit render. This guard converges after one update.
  if (hasFocusedTool && !showAllTools) {
    setShowAllTools(true);
  }

  return (
    <Paper
      data-part="block"
      sx={{
        mb: 1.5,
        overflow: "hidden",
        border: "1px solid",
        borderColor: isExpanded ? "primary.dark" : "divider",
        transition: "border-color 0.15s",
      }}
    >
      <BlockHeader
        block={block}
        isExpanded={isExpanded}
        onToggle={() => {
          if (isControlled) {
            onToggleExpanded?.();
            return;
          }
          setInternalExpanded((current) => !current);
        }}
      />

      <Collapse in={isExpanded} unmountOnExit>
        <Box sx={{ contentVisibility: "auto", containIntrinsicSize: "600px" }}>
          <BlockBody
            block={block}
            focusedTarget={focusedTarget}
            turnAnnotations={turnAnnotations}
            toolCallAnnotations={toolCallAnnotations}
            onCreateScopedAnnotation={onCreateScopedAnnotation}
            onOpenAnnotation={onOpenAnnotation}
            showAllTools={showAllTools}
            onShowAllTools={() => setShowAllTools(true)}
          />
        </Box>
      </Collapse>
    </Paper>
  );
}

export const BlockCard = memo(BlockCardImpl);
