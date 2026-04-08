import { memo, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Collapse from "@mui/material/Collapse";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
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
  onOpenAnnotation?: (annotation: Annotation) => void;
}

/** Simple line-based diff: color removed lines red, added lines green */
function DiffView({ oldText, newText }: { oldText: string; newTitle?: string; newText: string }) {
  const oldLines = oldText.split("\n");
  const newLines = newText.split("\n");
  return (
    <Box
      component="pre"
      sx={{
        m: 0,
        p: 1,
        bgcolor: "#0d1117",
        borderRadius: 1,
        overflow: "auto",
        maxHeight: 300,
        fontSize: "0.7rem",
        lineHeight: 1.5,
        fontFamily: "monospace",
      }}
    >
      {oldLines.map((line, i) => (
        <Box key={`old-${i}`} component="span" sx={{ color: "#f85149" }}>
          {"- "}{line}{"\n"}
        </Box>
      ))}
      {newLines.map((line, i) => (
        <Box key={`new-${i}`} component="span" sx={{ color: "#3fb950" }}>
          {"+ "}{line}{"\n"}
        </Box>
      ))}
    </Box>
  );
}

/** Scrollable code block for write tool call content */
function ContentBlock({ content }: { content: string }) {
  const [truncated, setTruncated] = useState(content.length > 2000);
  const displayContent = truncated ? content.slice(0, 2000) : content;
  return (
    <Box>
      <Box
        component="pre"
        sx={{
          m: 0,
          p: 1,
          bgcolor: "#0d1117",
          borderRadius: 1,
          overflow: "auto",
          maxHeight: 300,
          fontSize: "0.7rem",
          lineHeight: 1.5,
          fontFamily: "monospace",
          color: "success.light",
        }}
      >
        {displayContent}
        {truncated && "\n… (truncated)"}
      </Box>
      {truncated && (
        <Button
          size="small"
          onClick={() => setTruncated(false)}
          sx={{ mt: 0.5, fontSize: "0.65rem" }}
        >
          Show all ({content.length.toLocaleString()} chars)
        </Button>
      )}
    </Box>
  );
}

/** Render the expanded detail section, specialized by tool type */
function ToolCallDetail({ tc, cmd }: { tc: ToolCall; cmd: string }) {
  const args = tc.input.arguments;

  // Edit tool: show diffs
  if (tc.tool_name === "edit" && args?.edits && Array.isArray(args.edits)) {
    const edits = args.edits as Array<{ oldText?: string; newText?: string }>;
    return (
      <>
        {edits.length > 1 && (
          <Typography variant="caption" sx={{ fontFamily: "monospace", color: "text.secondary" }}>
            {edits.length} edit(s)
          </Typography>
        )}
        {edits.map((edit, i) => (
          <Box key={i} sx={{ mb: 1 }}>
            {edits.length > 1 && (
              <Typography variant="overline" color="text.secondary" sx={{ fontSize: "0.6rem" }}>
                Edit {i + 1}
              </Typography>
            )}
            <DiffView oldText={edit.oldText ?? ""} newText={edit.newText ?? ""} />
          </Box>
        ))}
        {tc.output.result && (
          <>
            <Typography variant="overline" color="text.secondary">
              Result
            </Typography>
            <Typography variant="caption" sx={{ fontFamily: "monospace", color: "text.secondary", display: "block" }}>
              {tc.output.result}
            </Typography>
          </>
        )}
      </>
    );
  }

  // Write tool: show content
  if (tc.tool_name === "write" && args?.content && typeof args.content === "string") {
    return (
      <>
        <ContentBlock content={args.content} />
        {tc.output.result && (
          <Typography variant="caption" sx={{ fontFamily: "monospace", color: "text.secondary", display: "block", mt: 0.5 }}>
            {tc.output.result}
          </Typography>
        )}
      </>
    );
  }

  // Bash tool: show command + output
  if (tc.tool_name === "bash") {
    return (
      <>
        <Typography variant="overline" color="text.secondary">
          Command
        </Typography>
        <Box
          component="pre"
          sx={{
            m: 0, mb: 1, p: 1,
            bgcolor: "#0d1117", borderRadius: 1,
            overflow: "auto", maxHeight: 200,
            whiteSpace: "pre-wrap", wordBreak: "break-all",
            fontSize: "0.7rem", color: "primary.light",
          }}
        >
          {cmd}
        </Box>
        {tc.output.result && (
          <>
            <Typography variant="overline" color="text.secondary">Output</Typography>
            <Box
              component="pre"
              sx={{
                m: 0, p: 1,
                bgcolor: "#0d1117", borderRadius: 1,
                overflow: "auto", maxHeight: 300,
                whiteSpace: "pre-wrap", wordBreak: "break-all",
                fontSize: "0.7rem", color: "success.light",
              }}
            >
              {tc.output.result}
            </Box>
          </>
        )}
        {tc.output.error && (
          <>
            <Typography variant="overline" color="error.main">Error</Typography>
            <Box
              component="pre"
              sx={{
                m: 0, p: 1,
                bgcolor: "#0d1117", borderRadius: 1,
                overflow: "auto", maxHeight: 300,
                whiteSpace: "pre-wrap", wordBreak: "break-all",
                fontSize: "0.7rem", color: "error.light",
              }}
            >
              {tc.output.error}
            </Box>
          </>
        )}
      </>
    );
  }

  // Generic fallback: command + output
  return (
    <>
      <Typography variant="overline" color="text.secondary">Command</Typography>
      <Box
        component="pre"
        sx={{
          m: 0, mb: 1, p: 1,
          bgcolor: "#0d1117", borderRadius: 1,
          overflow: "auto", maxHeight: 200,
          whiteSpace: "pre-wrap", wordBreak: "break-all",
          fontSize: "0.7rem", color: "primary.light",
        }}
      >
        {cmd}
      </Box>
      {tc.output.result && (
        <>
          <Typography variant="overline" color="text.secondary">Output</Typography>
          <Box
            component="pre"
            sx={{
              m: 0, p: 1,
              bgcolor: "#0d1117", borderRadius: 1,
              overflow: "auto", maxHeight: 300,
              whiteSpace: "pre-wrap", wordBreak: "break-all",
              fontSize: "0.7rem", color: "success.light",
            }}
          >
            {tc.output.result}
          </Box>
        </>
      )}
      {tc.output.error && (
        <>
          <Typography variant="overline" color="error.main">Error</Typography>
          <Box
            component="pre"
            sx={{
              m: 0, p: 1,
              bgcolor: "#0d1117", borderRadius: 1,
              overflow: "auto", maxHeight: 300,
              whiteSpace: "pre-wrap", wordBreak: "break-all",
              fontSize: "0.7rem", color: "error.light",
            }}
          >
            {tc.output.error}
          </Box>
        </>
      )}
    </>
  );
}

function ToolCallRowImpl({
  tc,
  defaultExpanded = false,
  focused = false,
  annotations = [],
  onAnnotate,
  onOpenAnnotation,
}: ToolCallRowProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  // Build a meaningful summary for the collapsed tool call row.
  // Priority: command (bash) > file_path (read/write/edit) > query (web_search) > tool_name
  const cmd =
    tc.input.command ||
    tc.input.file_path ||
    (tc.input.arguments?.query?.toString()
      ? `"${tc.input.arguments.query.toString()}"`
      : undefined) ||
    tc.input.arguments?.cmd?.toString() ||
    tc.input.arguments?.path?.toString() ||
    tc.input.arguments?.url?.toString() ||
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
            <Tooltip
              key={ann.id}
              title={`${ann.content.title}${ann.content.detail ? ` — ${ann.content.detail}` : ""}`}
              arrow
            >
              <Chip
                label={annotations.length === 1 ? ann.content.category : `${annotations.length} annotations`}
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
      <Collapse in={expanded} unmountOnExit>
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
          <ToolCallDetail tc={tc} cmd={cmd} />
        </Box>
      </Collapse>
    </Box>
  );
}

export const ToolCallRow = memo(ToolCallRowImpl);
