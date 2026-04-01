import { useState } from "react";
import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import Collapse from "@mui/material/Collapse";
import IconButton from "@mui/material/IconButton";
import Button from "@mui/material/Button";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import PersonIcon from "@mui/icons-material/Person";
import SmartToyIcon from "@mui/icons-material/SmartToy";
import CommitIcon from "@mui/icons-material/Commit";
import ConfirmationNumberIcon from "@mui/icons-material/ConfirmationNumber";
import EditNoteIcon from "@mui/icons-material/EditNote";
import type { SessionBlock } from "../../types";
import { ToolCallRow } from "./ToolCallRow";

interface BlockCardProps {
  block: SessionBlock;
  defaultExpanded?: boolean;
}

export function BlockCard({ block, defaultExpanded = false }: BlockCardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const [showAllTools, setShowAllTools] = useState(false);

  const date = new Date(block.user_ts);
  const timeStr = date.toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
  });
  const dateStr = date.toLocaleDateString("en-CA");

  const hasArtifacts =
    block.artifacts.commits.length > 0 ||
    block.artifacts.tickets_created.length > 0 ||
    block.artifacts.docs_added.length > 0 ||
    block.artifacts.diary_writes > 0;

  return (
    <Paper
      data-part="block"
      sx={{
        mb: 1.5,
        overflow: "hidden",
        border: "1px solid",
        borderColor: expanded ? "primary.dark" : "divider",
        transition: "border-color 0.15s",
      }}
    >
      {/* Block header */}
      <Box
        onClick={() => setExpanded(!expanded)}
        sx={{
          display: "flex",
          alignItems: "center",
          gap: 1.5,
          px: 2,
          py: 1,
          cursor: "pointer",
          bgcolor: expanded ? "rgba(245,166,35,0.04)" : "transparent",
          "&:hover": { bgcolor: "action.hover" },
        }}
      >
        <IconButton size="small" sx={{ p: 0 }}>
          <ExpandMoreIcon
            sx={{
              fontSize: 18,
              transform: expanded ? "rotate(0deg)" : "rotate(-90deg)",
              transition: "transform 0.15s",
            }}
          />
        </IconButton>

        <Chip
          label={`#${block.block_num}`}
          size="small"
          variant="outlined"
          sx={{ fontFamily: "monospace", fontWeight: 700, minWidth: 40 }}
        />

        <Typography variant="caption" sx={{ fontFamily: "monospace", color: "text.secondary" }}>
          {dateStr} {timeStr}
        </Typography>

        {block.gap_minutes != null && block.gap_minutes > 30 && (
          <Chip
            label={`${block.gap_minutes >= 60 ? (block.gap_minutes / 60).toFixed(1) + "h" : Math.round(block.gap_minutes) + "m"} gap`}
            size="small"
            color="warning"
            variant="outlined"
            sx={{ fontFamily: "monospace", fontSize: "0.6875rem" }}
          />
        )}

        <Typography
          variant="body2"
          sx={{
            flex: 1,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            fontWeight: expanded ? 600 : 400,
          }}
        >
          {block.user_content}
        </Typography>

        <Stack direction="row" spacing={1} alignItems="center">
          {block.artifacts.commits.length > 0 && (
            <Chip
              icon={<CommitIcon sx={{ fontSize: 14 }} />}
              label={block.artifacts.commits.length}
              size="small"
              color="success"
              variant="outlined"
              sx={{ fontFamily: "monospace" }}
            />
          )}
          {block.artifacts.tickets_created.length > 0 && (
            <Chip
              icon={<ConfirmationNumberIcon sx={{ fontSize: 14 }} />}
              label={block.artifacts.tickets_created.length}
              size="small"
              color="info"
              variant="outlined"
              sx={{ fontFamily: "monospace" }}
            />
          )}
          {block.artifacts.diary_writes > 0 && (
            <Chip
              icon={<EditNoteIcon sx={{ fontSize: 14 }} />}
              label={block.artifacts.diary_writes}
              size="small"
              color="warning"
              variant="outlined"
              sx={{ fontFamily: "monospace" }}
            />
          )}
          <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
            {block.agent_turns}t / {block.tool_calls}tc
          </Typography>
        </Stack>
      </Box>

      {/* Expanded content */}
      <Collapse in={expanded}>
        <Box sx={{ px: 2, pb: 2 }}>
          {/* Artifact summary */}
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

          {/* Turns */}
          {block.turns.map((t) => (
            <Box key={t.idx} sx={{ mb: 1.5 }}>
              {/* Role header */}
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
              </Stack>

              {/* Content */}
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

              {/* Tool calls */}
              {t.tool_calls_in_turn.length > 0 && (
                <Box sx={{ ml: 2 }}>
                  {(showAllTools
                    ? t.tool_calls_in_turn
                    : t.tool_calls_in_turn.slice(0, 5)
                  ).map((tc) => (
                    <ToolCallRow key={tc.id} tc={tc} />
                  ))}
                  {!showAllTools && t.tool_calls_in_turn.length > 5 && (
                    <Button
                      size="small"
                      onClick={() => setShowAllTools(true)}
                      sx={{ ml: 2, mt: 0.5, fontSize: "0.75rem" }}
                    >
                      Show all {t.tool_calls_in_turn.length} tool calls
                    </Button>
                  )}
                </Box>
              )}
            </Box>
          ))}
        </Box>
      </Collapse>
    </Paper>
  );
}
