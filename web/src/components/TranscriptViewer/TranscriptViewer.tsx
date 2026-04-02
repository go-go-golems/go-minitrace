import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import QueryStatsIcon from "@mui/icons-material/QueryStats";
import type { SessionDetail } from "../../types";
import { ActiveBadge, FormatWallActive } from "../shared";
import { BlockCard } from "./BlockCard";

interface TranscriptViewerProps {
  session: SessionDetail;
  onBack: () => void;
  onQuerySession: (id: string) => void;
}

export function TranscriptViewer({
  session,
  onBack,
  onQuerySession,
}: TranscriptViewerProps) {
  const activePct =
    (session.timing.active_duration_seconds /
      Math.max(session.timing.duration_seconds, 1)) *
    100;

  return (
    <Box
      data-widget="transcript-viewer"
      sx={{ height: "100%", display: "flex", flexDirection: "column" }}
    >
      {/* Header */}
      <Box sx={{ p: 2, display: "flex", alignItems: "center", gap: 2 }}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={onBack}
          size="small"
          variant="text"
        >
          Sessions
        </Button>
        <Typography
          variant="caption"
          sx={{ fontFamily: "monospace", color: "text.secondary" }}
        >
          {session.id.slice(0, 8)}
        </Typography>
        <Typography variant="h3" sx={{ flex: 1 }}>
          {session.title.slice(0, 100)}
        </Typography>
        <Button
          startIcon={<QueryStatsIcon />}
          onClick={() => onQuerySession(session.id)}
          size="small"
          variant="outlined"
        >
          Query
        </Button>
      </Box>

      {/* Session info bar */}
      <Paper sx={{ mx: 2, mb: 2, p: 1.5 }}>
        <Stack direction="row" spacing={3} alignItems="center" flexWrap="wrap">
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" color="text.secondary">
              Started
            </Typography>
            <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
              {new Date(session.timing.started_at).toLocaleString()}
            </Typography>
          </Stack>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" color="text.secondary">
              Duration
            </Typography>
            <FormatWallActive
              wallSeconds={session.timing.duration_seconds}
              activeSeconds={session.timing.active_duration_seconds}
            />
          </Stack>
          <ActiveBadge activePct={activePct} />
          <Chip
            label={`${session.metrics.turn_count} turns`}
            size="small"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
          <Chip
            label={`${session.metrics.tool_call_count} tool calls`}
            size="small"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
          <Chip
            label={session.environment.model}
            size="small"
            color="secondary"
            variant="outlined"
          />
          <Typography
            variant="caption"
            sx={{ fontFamily: "monospace", color: "text.secondary" }}
          >
            {session.operational_context.working_directory}
          </Typography>
        </Stack>
      </Paper>

      {/* Block list */}
      <Box sx={{ flex: 1, overflow: "auto", px: 2, pb: 2 }}>
        <Typography variant="overline" color="text.secondary" sx={{ mb: 1 }}>
          {session.blocks.length} blocks
        </Typography>
        {session.blocks.map((block) => (
          <BlockCard
            key={block.block_num}
            block={block}
            defaultExpanded={block.block_num === 1}
          />
        ))}
      </Box>
    </Box>
  );
}
