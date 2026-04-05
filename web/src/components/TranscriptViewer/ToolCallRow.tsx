import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Collapse from "@mui/material/Collapse";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import Stack from "@mui/material/Stack";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import CheckCircleOutlineIcon from "@mui/icons-material/CheckCircleOutline";
import ErrorOutlineIcon from "@mui/icons-material/ErrorOutline";
import BuildIcon from "@mui/icons-material/Build";
import type { Annotation, ToolCall } from "../../types";
import { ANNOTATION_CATEGORY_COLORS as CATEGORY_COLORS } from "../../types/session";
import { ToolCallBadgeChip } from "../shared";

interface ToolCallRowProps {
  tc: ToolCall;
  defaultExpanded?: boolean;
  focused?: boolean;
  annotations?: Annotation[];
  onAnnotate?: () => void;
}

export function ToolCallRow({
  tc,
  defaultExpanded = false,
  focused = false,
  annotations = [],
  onAnnotate,
}: ToolCallRowProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const cmd =
    tc.input.command ||
    tc.input.arguments?.cmd?.toString() ||
    tc.tool_name;

  return (
    <Box
      data-part="tool-call"
      data-tool-call-id={tc.id}
      sx={{
        borderLeft: "2px solid",
        borderColor: focused
          ? "warning.main"
          : tc.output.success
            ? "divider"
            : "error.main",
        bgcolor: focused ? "rgba(245,166,35,0.08)" : "transparent",
        borderRadius: 1,
        ml: 2,
        my: 0.5,
        transition: "background-color 0.2s, border-color 0.2s",
      }}
    >
      {/* Summary row */}
      <Box
        onClick={() => setExpanded(!expanded)}
        sx={{
          display: "flex",
          alignItems: "center",
          gap: 1,
          px: 1.5,
          py: 0.5,
          cursor: "pointer",
          borderRadius: 1,
          "&:hover": { bgcolor: "action.hover" },
        }}
      >
        <IconButton size="small" sx={{ p: 0 }}>
          <ExpandMoreIcon
            sx={{
              fontSize: 16,
              transform: expanded ? "rotate(0deg)" : "rotate(-90deg)",
              transition: "transform 0.15s",
            }}
          />
        </IconButton>
        <BuildIcon sx={{ fontSize: 14, color: "text.secondary" }} />
        <Typography
          variant="caption"
          sx={{ fontFamily: "monospace", color: "text.secondary", fontWeight: 600 }}
        >
          {tc.tool_name}
        </Typography>
        <Typography
          variant="caption"
          sx={{
            fontFamily: "monospace",
            color: "text.secondary",
            flex: 1,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {cmd.length > 80 ? cmd.slice(0, 80) + "…" : cmd}
        </Typography>
        <Stack direction="row" spacing={0.5}>
          {tc.badges.map((b) => (
            <ToolCallBadgeChip key={b} badge={b} />
          ))}
          {annotations.slice(0, 1).map((ann) => (
            <Chip
              key={ann.id}
              label={annotations.length === 1 ? ann.content.category : `${annotations.length} annotations`}
              size="small"
              color={CATEGORY_COLORS[ann.content.category] ?? "default"}
              variant="outlined"
              sx={{ height: 20, fontSize: "0.65rem" }}
            />
          ))}
        </Stack>
        <Button
          size="small"
          variant="text"
          sx={{ minWidth: 0, px: 0.75, fontSize: "0.7rem" }}
          onClick={(e) => {
            e.stopPropagation();
            onAnnotate?.();
          }}
        >
          Annotate
        </Button>
        <Typography variant="caption" sx={{ fontFamily: "monospace", opacity: 0.6, minWidth: 50, textAlign: "right" }}>
          {(tc.output.duration_ms / 1000).toFixed(1)}s
        </Typography>
        {tc.output.success ? (
          <CheckCircleOutlineIcon sx={{ fontSize: 14, color: "success.main" }} />
        ) : (
          <ErrorOutlineIcon sx={{ fontSize: 14, color: "error.main" }} />
        )}
      </Box>

      {/* Expanded detail */}
      <Collapse in={expanded}>
        <Box
          sx={{
            mx: 1.5,
            mb: 1,
            p: 1.5,
            bgcolor: "background.default",
            borderRadius: 1,
            fontFamily: "monospace",
            fontSize: "0.75rem",
            lineHeight: 1.6,
          }}
        >
          <Typography variant="overline" color="text.secondary">
            Command
          </Typography>
          <Box
            component="pre"
            sx={{
              m: 0,
              mb: 1,
              p: 1,
              bgcolor: "#0d1117",
              borderRadius: 1,
              overflow: "auto",
              maxHeight: 200,
              whiteSpace: "pre-wrap",
              wordBreak: "break-all",
              fontSize: "0.75rem",
              color: "primary.light",
            }}
          >
            {cmd}
          </Box>
          {tc.output.result && (
            <>
              <Typography variant="overline" color="text.secondary">
                Output
              </Typography>
              <Box
                component="pre"
                sx={{
                  m: 0,
                  p: 1,
                  bgcolor: "#0d1117",
                  borderRadius: 1,
                  overflow: "auto",
                  maxHeight: 300,
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-all",
                  fontSize: "0.75rem",
                  color: "success.light",
                }}
              >
                {tc.output.result}
              </Box>
            </>
          )}
          {tc.output.error && (
            <>
              <Typography variant="overline" color="error.main">
                Error
              </Typography>
              <Box
                component="pre"
                sx={{
                  m: 0,
                  p: 1,
                  bgcolor: "#0d1117",
                  borderRadius: 1,
                  overflow: "auto",
                  maxHeight: 300,
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-all",
                  fontSize: "0.75rem",
                  color: "error.light",
                }}
              >
                {tc.output.error}
              </Box>
            </>
          )}
        </Box>
      </Collapse>
    </Box>
  );
}
