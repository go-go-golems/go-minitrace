import { memo, useEffect, useState } from "react";
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
  forceExpanded?: boolean;
  focusedTarget?: FocusedTranscriptTarget | null;
  turnAnnotations?: Record<string, Annotation[]>;
  toolCallAnnotations?: Record<string, Annotation[]>;
  onCreateScopedAnnotation?: (
    scopeType: "session" | "turn" | "tool_call",
    targetId: string,
  ) => void;
  onOpenAnnotation?: (annotation: Annotation) => void;
}

function BlockCardImpl({
  block,
  defaultExpanded = false,
  forceExpanded = false,
  focusedTarget = null,
  turnAnnotations = {},
  toolCallAnnotations = {},
  onCreateScopedAnnotation,
  onOpenAnnotation,
}: BlockCardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const [showAllTools, setShowAllTools] = useState(false);
  const isExpanded = expanded || forceExpanded;

  useEffect(() => {
    if (
      focusedTarget?.scopeType === "tool_call" &&
      block.turns.some((t) =>
        t.tool_calls_in_turn.some((tc) => tc.id === focusedTarget.targetId),
      )
    ) {
      setShowAllTools(true);
    }
  }, [block.turns, focusedTarget]);

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
        onToggle={() => setExpanded(!isExpanded)}
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
