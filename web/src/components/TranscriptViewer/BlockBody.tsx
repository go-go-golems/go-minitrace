import { memo } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import PersonIcon from "@mui/icons-material/Person";
import SmartToyIcon from "@mui/icons-material/SmartToy";
import type { Annotation, SessionBlock } from "../../types";
import { ANNOTATION_CATEGORY_COLORS as CATEGORY_COLORS } from "../../types/session";
import { ToolCallRow } from "./ToolCallRow";
import type { FocusedTranscriptTarget } from "./types";

interface BlockBodyProps {
  block: SessionBlock;
  focusedTarget?: FocusedTranscriptTarget | null;
  turnAnnotations?: Record<string, Annotation[]>;
  toolCallAnnotations?: Record<string, Annotation[]>;
  onCreateScopedAnnotation?: (
    scopeType: "session" | "turn" | "tool_call",
    targetId: string,
  ) => void;
  onOpenAnnotation?: (annotation: Annotation) => void;
  showAllTools: boolean;
  onShowAllTools: () => void;
}

function BlockBodyImpl({
  block,
  focusedTarget = null,
  turnAnnotations = {},
  toolCallAnnotations = {},
  onCreateScopedAnnotation,
  onOpenAnnotation,
  showAllTools,
  onShowAllTools,
}: BlockBodyProps) {
  const hasArtifacts =
    block.artifacts.commits.length > 0 ||
    block.artifacts.tickets_created.length > 0 ||
    block.artifacts.docs_added.length > 0 ||
    block.artifacts.diary_writes > 0;

  return (
    <Box sx={{ px: 2, pb: 2 }}>
      {hasArtifacts && (
        <Box sx={{ mb: 2, p: 1.5, bgcolor: "background.default", borderRadius: 1 }}>
          {block.artifacts.tickets_created.map((t) => (
            <Typography
              key={t}
              variant="caption"
              sx={{ display: "block", color: "info.main", fontFamily: "monospace" }}
            >
              📋 Ticket created: {t}
            </Typography>
          ))}
          {block.artifacts.docs_added.map((d) => (
            <Typography
              key={d}
              variant="caption"
              sx={{ display: "block", color: "secondary.main", fontFamily: "monospace" }}
            >
              📄 Doc added: {d}
            </Typography>
          ))}
          {block.artifacts.commits.map((c) => (
            <Typography
              key={c}
              variant="caption"
              sx={{ display: "block", color: "success.main", fontFamily: "monospace" }}
            >
              ✅ Commit: {c}
            </Typography>
          ))}
        </Box>
      )}

      {block.turns.map((t) => {
        const turnFocused =
          focusedTarget?.scopeType === "turn" &&
          focusedTarget.targetId === String(t.idx);

        return (
          <Box
            key={t.idx}
            data-turn-idx={String(t.idx)}
            sx={{
              mb: 1.5,
              px: 1,
              py: 0.5,
              borderRadius: 1,
              border: "1px solid",
              borderColor: turnFocused ? "warning.main" : "transparent",
              bgcolor: turnFocused ? "rgba(245,166,35,0.08)" : "transparent",
              transition: "background-color 0.2s, border-color 0.2s",
            }}
          >
            <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
              {t.role === "user" ? (
                <PersonIcon sx={{ fontSize: 16, color: "primary.main" }} />
              ) : (
                <SmartToyIcon sx={{ fontSize: 16, color: "secondary.main" }} />
              )}
              <Typography
                variant="overline"
                sx={{
                  color: t.role === "user" ? "primary.main" : "secondary.main",
                }}
              >
                {t.role}
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
                #{t.idx}
              </Typography>
              <Button
                size="small"
                variant="text"
                sx={{ minWidth: 0, px: 0.75, fontSize: "0.7rem" }}
                onClick={() => onCreateScopedAnnotation?.("turn", String(t.idx))}
              >
                Annotate
              </Button>
              {(turnAnnotations[String(t.idx)] ?? []).slice(0, 2).map((ann) => (
                <Tooltip
                  key={ann.id}
                  title={`${ann.content.title}${ann.content.detail ? ` — ${ann.content.detail}` : ""}`}
                  arrow
                >
                  <Chip
                    label={ann.content.category}
                    size="small"
                    color={CATEGORY_COLORS[ann.content.category] ?? "default"}
                    variant="outlined"
                    onClick={(e) => {
                      e.stopPropagation();
                      onOpenAnnotation?.(ann);
                    }}
                    sx={{ height: 20, fontSize: "0.65rem", cursor: "pointer" }}
                  />
                </Tooltip>
              ))}
              {(turnAnnotations[String(t.idx)] ?? []).length > 2 && (
                <Chip
                  label={`+${(turnAnnotations[String(t.idx)] ?? []).length - 2}`}
                  size="small"
                  variant="outlined"
                  sx={{ height: 20, fontSize: "0.65rem" }}
                />
              )}
            </Stack>

            {t.content && (
              <Typography
                variant="body2"
                sx={{
                  ml: 3,
                  mb: 1,
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-word",
                  lineHeight: 1.65,
                }}
              >
                {t.content}
              </Typography>
            )}

            {t.tool_calls_in_turn.length > 0 && (
              <Box sx={{ ml: 2 }}>
                {(showAllTools ? t.tool_calls_in_turn : t.tool_calls_in_turn.slice(0, 5)).map((tc) => (
                  <ToolCallRow
                    key={tc.id}
                    tc={tc}
                    focused={
                      focusedTarget?.scopeType === "tool_call" &&
                      focusedTarget.targetId === tc.id
                    }
                    annotations={toolCallAnnotations[tc.id] ?? []}
                    onAnnotate={() => onCreateScopedAnnotation?.("tool_call", tc.id)}
                    onOpenAnnotation={onOpenAnnotation}
                  />
                ))}
                {!showAllTools && t.tool_calls_in_turn.length > 5 && (
                  <Button
                    size="small"
                    onClick={onShowAllTools}
                    sx={{ ml: 2, mt: 0.5, fontSize: "0.75rem" }}
                  >
                    Show all {t.tool_calls_in_turn.length} tool calls
                  </Button>
                )}
              </Box>
            )}
          </Box>
        );
      })}
    </Box>
  );
}

export const BlockBody = memo(BlockBodyImpl);
